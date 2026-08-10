package itemTypes

type ListItemReq struct {
}
type ListItemRsp struct {
	List []Item `json:"list"`
}
