package player

import pbPlayerActorRPC "github.com/2comjie/taoxi-server/pb/player/actor_rpc"

func (p *Player) GetState(_ *pbPlayerActorRPC.GetStateReq, rsp *pbPlayerActorRPC.GetStateRsp) error {
	state := p.State()
	rsp.Uid = state.GetUid()
	rsp.LastOfflineTime = state.GetLastOfflineTime()
	return nil
}
