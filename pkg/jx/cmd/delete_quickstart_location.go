package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	deleteQuickstartLocationLong = templates.LongDesc(`
		Deletes one or more quickstart locations for your team

		For more documentation see: [https://jenkins-x.io/developing/create-quickstart/#customising-your-teams-quickstarts](https://jenkins-x.io/developing/create-quickstart/#customising-your-teams-quickstarts)

`)

	deleteQuickstartLocationExample = templates.Examples(`
		# Pick a quickstart location to delete for your team
		jx delete quickstartlocation

		# Pick a quickstart location to delete for your team using an abbreviation
		jx delete qsloc
	
		# Delete a github organisation 'myorg' for your team
		jx delete qsloc --owner myorg
		
		# Delete a specific location for your team
		jx delete qsloc --url https://foo.com --owner myowner

	`)
)

// DeleteQuickstartLocationOptions the options for the create spring command
type DeleteQuickstartLocationOptions struct {
	CommonOptions

	GitUrl string
	Owner  string
}

// NewCmdDeleteQuickstartLocation defines the command
func NewCmdDeleteQuickstartLocation(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteQuickstartLocationOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     quickstartLocation,
		Short:   "Deletes one or more quickstart locations for your team",
		Aliases: quickstartLocationsAliases,
		Long:    deleteQuickstartLocationLong,
		Example: deleteQuickstartLocationExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.GitUrl, optionGitUrl, "u", gits.GitHubURL, "The URL of the git service")
	cmd.Flags().StringVarP(&options.Owner, optionOwner, "o", "", "The owner is the user or organisation of the git provider")

	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *DeleteQuickstartLocationOptions) Run() error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	err = o.registerEnvironmentCRD()
	if err != nil {
		return err
	}

	locations, err := kube.GetQuickstartLocations(jxClient, ns)
	if err != nil {
		return err
	}

	if o.GitUrl == "" || o.Owner == "" {
		if o.BatchMode {
			if o.GitUrl == "" {
				return util.MissingOption(optionGitUrl)
			}
			if o.Owner == "" {
				return util.MissingOption(optionOwner)
			}
		} else {
			// TODO lets pick the available combinations to delete
			names := []string{}
			m := map[string]v1.QuickStartLocation{}
			for _, loc := range locations {
				key := util.UrlJoin(loc.GitURL, loc.Owner)
				m[key] = loc
				names = append(names, key)
			}

			name, err := util.PickName(names, "Pick the quickstart git owner to remove from the team settings: ")
			if err != nil {
				return err
			}
			if name == "" {
				return fmt.Errorf("No owner name chosen")
			}
			loc := m[name]
			o.Owner = loc.Owner
			o.GitUrl = loc.GitURL
		}
	}

	callback := func(env *v1.Environment) error {
		settings := &env.Spec.TeamSettings
		for i, l := range settings.QuickstartLocations {
			if l.GitURL == o.GitUrl && l.Owner == o.Owner {
				settings.QuickstartLocations = append(settings.QuickstartLocations[0:i], settings.QuickstartLocations[i+1:]...)
				o.Printf("Removing quickstart git owner %s\n", util.ColorInfo(util.UrlJoin(l.GitURL, l.Owner)))
				return nil
			}
		}
		return fmt.Errorf("No quickstart location found for git URL: %s and owner: %s", o.GitUrl, o.Owner)
	}
	return o.modifyDevEnvironment(jxClient, ns, callback)
}
