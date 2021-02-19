package state

import (
	"fmt"

	"github.com/oslokommune/okctl/pkg/api"

	"github.com/oslokommune/okctl/pkg/client"
	"github.com/oslokommune/okctl/pkg/client/store"
	"github.com/oslokommune/okctl/pkg/config/state"
)

type kubePromStackState struct {
	state state.RepositoryStateWithEnv
}

func (s *kubePromStackState) RemoveKubePromStack(id api.ID) (*store.Report, error) {
	m := s.state.GetMonitoring()
	m.DashboardURL = ""

	return s.state.SaveMonitoring(m)
}

func (s *kubePromStackState) SaveKubePromStack(kube *client.KubePromStack) (*store.Report, error) {
	m := s.state.GetMonitoring()
	m.DashboardURL = fmt.Sprintf("https://%s", kube.Hostname)

	return s.state.SaveMonitoring(m)
}

// NewKubePromStackState returns an initialised state layer
func NewKubePromStackState(state state.RepositoryStateWithEnv) client.KubePromStackState {
	return &kubePromStackState{
		state: state,
	}
}