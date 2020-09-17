package console

import (
	"fmt"
	"io"

	"github.com/logrusorgru/aurora"
	"github.com/oslokommune/okctl/pkg/client"
	"github.com/oslokommune/okctl/pkg/client/store"
	"github.com/theckman/yacspin"
)

type externalDNSReport struct {
	console *Console
}

func (r *externalDNSReport) ReportCreateExternalDNS(secret *client.ExternalDNS, report *store.Report) error {
	description := fmt.Sprintf("%s (policy), %s (service account), %s (kubernetes)",
		aurora.Green(secret.Policy.StackName),
		aurora.Green(secret.ServiceAccount.Config.Metadata.Name),
		aurora.Green("external-dns"),
	)

	return r.console.Report(report.Actions, "external-dns", description)
}

// NewExternalDNSReport returns an initialised reporter
func NewExternalDNSReport(out io.Writer, exit chan struct{}, spinner *yacspin.Spinner) client.ExternalDNSReport {
	return &externalDNSReport{
		console: New(out, exit, spinner),
	}
}