package diff_manager

import (
	"context"

	"github.com/2comjie/nova/diff"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type Manager[Pb proto.Message] struct {
	redis        redis.UniversalClient
	key          string
	compactCount int64
	applyDiff    func(Pb, []byte) error
}

func New[Pb proto.Message](rc redis.UniversalClient, key string, compactCount int64, applyDiff func(Pb, []byte) error) *Manager[Pb] {
	return &Manager[Pb]{redis: rc, key: key, compactCount: compactCount, applyDiff: applyDiff}
}

func (m *Manager[Pb]) Load(ctx context.Context, value Pb) (Pb, uint64, error) {
	list, err := m.redis.LRange(ctx, m.key, 0, -1).Result()
	if err != nil {
		return value, 0, err
	}
	if len(list) == 0 {
		return value, 0, nil
	}

	full, err := diff.NewSyncReader([]byte(list[0]))
	if err != nil {
		return value, 0, err
	}
	if err = proto.Unmarshal(full.FullData(), value); err != nil {
		return value, 0, err
	}
	version := full.BaseVersion()

	for _, data := range list[1:] {
		reader, err := diff.NewSyncReader([]byte(data))
		if err != nil {
			return value, 0, err
		}
		for {
			delta, ok, err := reader.NextDiff()
			if err != nil {
				return value, 0, err
			}
			if !ok {
				break
			}
			if err = m.applyDiff(value, delta.Data); err != nil {
				return value, 0, err
			}
			version = delta.Version
		}
	}
	return value, version, nil
}

func (m *Manager[Pb]) Save(ctx context.Context, data []byte, buildSnapshot func() (uint64, []byte)) error {
	count, err := m.redis.LLen(ctx, m.key).Result()
	if err != nil {
		return err
	}
	if count > 0 && count < m.compactCount {
		return m.redis.RPush(ctx, m.key, data).Err()
	}

	version, snapshot := buildSnapshot()
	writer := diff.NewSyncWriter(nil)
	writer.WriteFull(version, snapshot, nil)
	_, err = m.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, m.key)
		pipe.RPush(ctx, m.key, writer.Data())
		return nil
	})
	return err
}
