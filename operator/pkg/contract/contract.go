package contract

type WatcherEvent struct {
	SkrClusterID string `json:"skrClusterID"`
	Namespace    string `json:"namespace""`
	Name         string `json:"name""`
}
