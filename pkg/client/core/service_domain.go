package core

import (
	"context"
	"io"

	"github.com/oslokommune/okctl/pkg/config/state"

	"github.com/oslokommune/okctl/pkg/domain"

	"github.com/oslokommune/okctl/pkg/client/store"

	"github.com/oslokommune/okctl/pkg/ask"

	"github.com/oslokommune/okctl/pkg/client"
)

type domainService struct {
	out    io.Writer
	ask    *ask.Ask
	api    client.DomainAPI
	store  client.DomainStore
	state  client.DomainState
	report client.DomainReport
}

func (s *domainService) DeletePrimaryHostedZone(_ context.Context, opts client.DeletePrimaryHostedZoneOpts) error {
	var hz *state.HostedZone

	for _, z := range s.state.GetHostedZones() {
		if z.Primary {
			hz = &z
		}
	}

	if hz == nil {
		// Couldn't find a primary hosted zone, which means it has
		// already been removed
		return nil
	}

	var reports []*store.Report

	if hz.Managed {
		// HostedZone is managed by us, so delete it
		err := s.api.DeletePrimaryHostedZone(hz.Domain, opts)
		if err != nil {
			return err
		}

		report, err := s.store.RemoveHostedZone(hz.Domain)
		if err != nil {
			return err
		}

		reports = append(reports, report)
	}

	report, err := s.state.RemoveHostedZone(hz.Domain)
	if err != nil {
		return err
	}

	err = s.report.ReportDeletePrimaryHostedZone(append([]*store.Report{report}, reports...))
	if err != nil {
		return err
	}

	return nil
}

func (s *domainService) CreatePrimaryHostedZone(_ context.Context, opts client.CreatePrimaryHostedZoneOpts) (*client.HostedZone, error) {
	for _, z := range s.state.GetHostedZones() {
		if z.Primary {
			return s.store.GetHostedZone(z.Domain)
		}
	}

	// Shouldn't be doing this in here I think
	d, err := s.ask.Domain(domain.Default(opts.ID.Repository, opts.ID.Environment))
	if err != nil {
		return nil, err
	}

	opts.Domain = d.Domain
	opts.FQDN = d.FQDN

	zone, err := s.api.CreatePrimaryHostedZone(opts)
	if err != nil {
		return nil, err
	}

	delegated, err := s.ask.ConfirmPostingNameServers(s.out, zone.HostedZone.Domain, zone.HostedZone.NameServers)
	if err != nil {
		return nil, err
	}

	zone.IsDelegated = delegated

	r1, err := s.store.SaveHostedZone(zone)
	if err != nil {
		return nil, err
	}

	r2, err := s.state.SaveHostedZone(zone)
	if err != nil {
		return nil, err
	}

	err = s.report.ReportCreatePrimaryHostedZone(zone, []*store.Report{r1, r2})
	if err != nil {
		return nil, err
	}

	return zone, nil
}

// NewDomainService returns an initialised service
func NewDomainService(
	out io.Writer,
	ask *ask.Ask,
	api client.DomainAPI,
	store client.DomainStore,
	report client.DomainReport,
	state client.DomainState,
) client.DomainService {
	return &domainService{
		api:    api,
		out:    out,
		store:  store,
		report: report,
		ask:    ask,
		state:  state,
	}
}
