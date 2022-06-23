package contract

type WatcherEvent struct {
	SkrClusterID string `json:"skrClusterID"`
	Component    string `json:"body"`
	Namespace    string `json:"namespace""`
	Name         string `json:"name""`
}
