package commands

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"text/template"

	"github.com/logrusorgru/aurora/v3"

	"github.com/oslokommune/okctl/pkg/apis/okctl.io/v1alpha1"

	"github.com/oslokommune/okctl/pkg/controller"
	"github.com/oslokommune/okctl/pkg/controller/reconciler"
	"github.com/oslokommune/okctl/pkg/controller/resourcetree"

	"github.com/oslokommune/okctl/pkg/config/constant"
	"github.com/spf13/afero"

	"sigs.k8s.io/yaml"
)

// SynchronizeApplicationOpts contains references necessary to synchronize an application
type SynchronizeApplicationOpts struct {
	ReconciliationManager reconciler.Reconciler
	Application           v1alpha1.Application

	Tree *resourcetree.ResourceNode
}

// SynchronizeApplication knows how to discover differences between desired and actual state and rectify them
func SynchronizeApplication(opts SynchronizeApplicationOpts) error {
	opts.Tree.ApplyFunction(applyDesiredState(opts.Application.Image), opts.Tree)

	return controller.HandleNode(opts.ReconciliationManager, opts.Tree)
}

func applyDesiredState(image v1alpha1.ApplicationImage) resourcetree.ApplyFn {
	return func(receiver *resourcetree.ResourceNode, target *resourcetree.ResourceNode) {
		switch receiver.Type {
		case resourcetree.ResourceNodeTypeContainerRepository:
			if image.HasName() {
				receiver.State = resourcetree.ResourceNodeStatePresent
			} else {
				receiver.State = resourcetree.ResourceNodeStateNoop
			}
		case resourcetree.ResourceNodeTypeApplication:
			receiver.State = resourcetree.ResourceNodeStatePresent
		}
	}
}

// InferApplicationFromStdinOrFile returns an okctl application based on input. The function will parse input either
// from the reader or from the fs based on if path is a path or if it is "-". "-" represents stdin
func InferApplicationFromStdinOrFile(declaration v1alpha1.Cluster, stdin io.Reader, fs *afero.Afero, path string) (v1alpha1.Application, error) {
	var (
		err         error
		inputReader io.Reader
		app         = v1alpha1.NewApplication(declaration)
	)

	switch path {
	case "-":
		inputReader = stdin
	default:
		inputReader, err = fs.Open(filepath.Clean(path))
		if err != nil {
			return app, fmt.Errorf("opening application file: %w", err)
		}
	}

	var buf []byte

	buf, err = ioutil.ReadAll(inputReader)
	if err != nil {
		return app, fmt.Errorf("reading application file: %w", err)
	}

	err = yaml.Unmarshal(buf, &app)
	if err != nil {
		return app, fmt.Errorf("parsing application yaml: %w", err)
	}

	return app, nil
}

// ApplyApplicationSuccessMessageOpts contains the values for customizing the apply application success message
type ApplyApplicationSuccessMessageOpts struct {
	ApplicationName           string
	OptionalDockerTagPushStep string
	OptionalDockerImageURI    string
	KubectlApplyArgoCmd       string
}

const applyApplicationSuccessMessage = `
	Successfully scaffolded {{ .ApplicationName }}
	To deploy your application:
		- Commit and push the changes done by okctl{{ .OptionalDockerTagPushStep }}
		- Run {{ .KubectlApplyArgoCmd }}

    If using an ingress, it can take up to five minutes for the routing to configure
`

// WriteApplyApplicationSuccessMessage produces a relevant message for successfully reconciling an application
func WriteApplyApplicationSuccessMessage(writer io.Writer, application v1alpha1.Application, outputDir string) error {
	argoCDResourcePath := path.Join(
		outputDir,
		constant.DefaultApplicationsOutputDir,
		application.Metadata.Name,
		"argocd-application.yaml",
	)

	optionalDockerTagPushStep := ""

	if application.Image.HasName() {
		optionalDockerTagPushStep = `
        - Tag and push a docker image to your container repository. See instructions on
          https://okctl.io/help/docker-registry/#push-a-docker-image-to-the-amazon-elastic-container-registry-ecr`
	}

	tmpl, err := template.New("t").Parse(applyApplicationSuccessMessage)
	if err != nil {
		return err
	}

	var tmplBuffer bytes.Buffer

	err = tmpl.Execute(&tmplBuffer, ApplyApplicationSuccessMessageOpts{
		ApplicationName:           application.Metadata.Name,
		OptionalDockerTagPushStep: optionalDockerTagPushStep,
		OptionalDockerImageURI:    application.Image.URI,
		KubectlApplyArgoCmd:       aurora.Green(fmt.Sprintf("kubectl apply -f %s", argoCDResourcePath)).String(),
	})
	if err != nil {
		return err
	}

	fmt.Fprint(writer, tmplBuffer.String())

	return nil
}