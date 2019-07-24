package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"k8s.io/helm/pkg/helm"
	helmenv "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/tlsutil"
)

const globalUsage = `
Check to see if there is an updated version available for installed charts.
`

var outputFormat string
var devel bool
var version = "canary"

var (
	settings 		helmenv.EnvSettings
)

const (
	statusOutdated = "OUTDATED"
	statusUptodate = "UPTODATE"
)

type ChartVersionInfo struct {
	ReleaseName      string `json:"releaseName"`
	ChartName        string `json:"chartName"`
	InstalledVersion string `json:"installedVersion"`
	LatestVersion    string `json:"latestVersion"`
	Status           string `json:"status"`
}

func main() {
	cmd := &cobra.Command{
		Use:   "whatup [flags]",
		Short: fmt.Sprintf("check if installed charts are out of date (helm-whatup %s)", version),
		RunE:  run,
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "plain", "Output format, choose from plain, json, yaml")
	cmd.Flags().BoolVarP(&devel, "devel", "d", false, "Whether to include pre-releases or not, defaults to false.")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	client := newClient()

	releases, err := fetchReleases(client)
	if err != nil {
		return err
	}

	repositories, err := fetchIndices(client)
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		if outputFormat == "plain" {
			fmt.Println("No releases found. All up to date!")
		}
		return nil
	}

	if len(repositories) == 0 {
		if outputFormat == "plain" {
			fmt.Println("No repositories found. Did you run `helm repo update`?")
		}
		return nil
	}

	var result []ChartVersionInfo

	for _, release := range releases {
		for _, idx := range repositories {
			if idx.Has(release.Chart.Metadata.Name, release.Chart.Metadata.Version) {
				// fetch latest release
				constraint := ""
				// Include pre-releases
				if devel {
					constraint = ">= *-0"
				}
				chartVer, err := idx.Get(release.Chart.Metadata.Name, constraint)
				if err != nil {
					return err
				}

				versionStatus := ChartVersionInfo{
					ReleaseName:      release.Name,
					ChartName:        release.Chart.Metadata.Name,
					InstalledVersion: release.Chart.Metadata.Version,
					LatestVersion:    chartVer.Version,
				}

				if versionStatus.InstalledVersion == versionStatus.LatestVersion {
					versionStatus.Status = statusUptodate
				} else {
					versionStatus.Status = statusOutdated
				}
				result = append(result, versionStatus)
			}
		}
	}

	switch outputFormat {
	case "plain":
		for _, versionInfo := range result {
			if versionInfo.LatestVersion != versionInfo.InstalledVersion {
				fmt.Printf("There is an update available for release %s (%s)!\nInstalled version: %s\nAvailable version: %s\n", versionInfo.ReleaseName, versionInfo.ChartName, versionInfo.InstalledVersion, versionInfo.LatestVersion)
			} else {
				fmt.Printf("Release %s (%s) is up to date.\n", versionInfo.ReleaseName, versionInfo.LatestVersion)
			}
		}
		fmt.Println("Done.")

	case "json":
		outputBytes, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			return err
		}
		fmt.Println(string(outputBytes))

	case "yml":
		fallthrough
	case "yaml":
		outputBytes, err := yaml.Marshal(result)
		if err != nil {
			return err
		}
		fmt.Println(string(outputBytes))

	default:
		return fmt.Errorf("invalid formatter: %s", outputFormat)
	}

	return nil
}


func newClient() *helm.Client {
	/// === Pre-Checks ===
	if settings.TillerHost == "" {
		if os.Getenv("TILLER_HOST") != "" {
			settings.TillerHost = os.Getenv("TILLER_HOST")

		} else if os.Getenv("HELM_HOST") != "" {
			settings.TillerHost = os.Getenv("HELM_HOST")
		}

		if settings.TillerHost == "" {
			fmt.Errorf("error: Tiller Host not set")
			os.Exit(1)
		}
	}

	if settings.TLSCaCertFile == helmenv.DefaultTLSCaCert || settings.TLSCaCertFile == "" {
		settings.TLSCaCertFile = settings.Home.TLSCaCert()
	} else {
		settings.TLSCaCertFile = os.ExpandEnv(settings.TLSCaCertFile)
	}

	if settings.TLSCertFile == helmenv.DefaultTLSCert || settings.TLSCertFile == "" {
		settings.TLSCertFile = settings.Home.TLSCert()
	} else {
		settings.TLSCertFile = os.ExpandEnv(settings.TLSCertFile)
	}

	if settings.TLSKeyFile == helmenv.DefaultTLSKeyFile || settings.TLSKeyFile == "" {
		settings.TLSKeyFile = settings.Home.TLSKey()
	} else {
		settings.TLSKeyFile = os.ExpandEnv(settings.TLSKeyFile)
	}

	options := []helm.Option{ helm.Host(settings.TillerHost) }

	// check if TLS is enabled
	if settings.TLSEnable || settings.TLSVerify {
		tlsopts := tlsutil.Options{
			ServerName:			settings.TillerHost,
			CaCertFile:			settings.TLSCaCertFile,
			CertFile:			settings.TLSCertFile,
			KeyFile:			settings.TLSKeyFile,
			InsecureSkipVerify:	!settings.TLSVerify,
		}

		tlscfg, err := tlsutil.ClientConfig(tlsopts)

		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		options = append(options, helm.WithTLS(tlscfg))
	}

	return helm.NewClient(options...)
}

func fetchReleases(client *helm.Client) ([]*release.Release, error) {
	res, err := client.ListReleases()

	if err != nil {
		return nil, err
	}

	if res == nil {
		return []*release.Release{}, nil
	}

	return res.Releases, nil
}

func fetchIndices(client *helm.Client) ([]*repo.IndexFile, error) {
	indices := []*repo.IndexFile{}
	rfp := os.Getenv("HELM_PATH_REPOSITORY_FILE")
	repofile, err := repo.LoadRepositoriesFile(rfp)

	if err != nil {
		return nil, fmt.Errorf("could not load repositories file '%s': %s", rfp, err)
	}

	for _, repository := range repofile.Repositories {
		idx, err := repo.LoadIndexFile(repository.Cache)
		if err != nil {
			return nil, fmt.Errorf("could not load index file '%s': %s", repository.Cache, err)
		}
		indices = append(indices, idx)
	}

	return indices, nil
}