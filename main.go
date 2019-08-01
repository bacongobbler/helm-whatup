package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/tlsutil"
)

const globalUsage = `
Check to see if there is an updated version available for installed charts.
`

var (
	outputFormat string
	settings     environment.EnvSettings
	devel        bool
	tlsEnable    bool
	tlsHostname  string
	tlsCaCert    string
	tlsCert      string
	tlsKey       string
	tlsVerify    bool
)

var version = "canary"

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

	f := cmd.Flags()

	f.StringVarP(&outputFormat, "output", "o", "plain", "output format. Accepted formats: plain, json, yaml, table")
	f.BoolVarP(&devel, "devel", "d", false, "whether to include pre-releases or not")
	f.BoolVar(&tlsEnable, "tls", false, "enable TLS for requests to the server")
	f.StringVar(&tlsCaCert, "tls-ca-cert", "", "path to TLS CA certificate file")
	f.StringVar(&tlsCert, "tls-cert", "", "path to TLS certificate file")
	f.StringVar(&tlsKey, "tls-key", "", "path to TLS key file")
	f.StringVar(&tlsHostname, "tls-hostname", "", "the server name used to verify the hostname on the returned certificates from the server")
	f.BoolVar(&tlsVerify, "tls-verify", false, "enable TLS for requests to the server, and controls whether the client verifies the server's certificate chain and host name")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newClient() (*helm.Client, error) {
	opts := []helm.Option{}

	helmHost := os.Getenv("TILLER_HOST")

	if helmHost == "" {
		return nil, fmt.Errorf("TILLER_HOST was not set by Helm")
	}

	opts = append(opts, helm.Host(helmHost))

	// priority order for reading in configuration:
	//
	// 1. flags
	// 2. environment variables
	// 3. defaults

	if tlsHostname == "" {
		tlsHostname = os.Getenv("HELM_TLS_HOSTNAME")
		if tlsHostname == "" {
			tlsHostname = helmHost
		}
	}

	if tlsCaCert == "" {
		tlsCaCert = os.Getenv("HELM_TLS_CA_CERT")
		if tlsCaCert == "" {
			tlsCaCert = os.ExpandEnv(environment.DefaultTLSCaCert)
		}
	}

	if tlsCert == "" {
		tlsCert = os.Getenv("HELM_TLS_CERT")
		if tlsCert == "" {
			tlsCert = os.ExpandEnv(environment.DefaultTLSCert)
		}
	}

	if tlsKey == "" {
		tlsKey = os.Getenv("HELM_TLS_KEY")
		if tlsKey == "" {
			tlsKey = os.ExpandEnv(environment.DefaultTLSKeyFile)
		}
	}

	if !tlsEnable {
		tlsEnable = os.Getenv("HELM_TLS_ENABLE") != ""
	}

	if !tlsVerify {
		tlsVerify = os.Getenv("HELM_TLS_VERIFY") != ""
	}

	if tlsEnable || tlsVerify {
		tlsopts := tlsutil.Options{
			ServerName:         tlsHostname,
			CaCertFile:         tlsCaCert,
			CertFile:           tlsCert,
			KeyFile:            tlsKey,
			InsecureSkipVerify: !tlsVerify,
		}

		tlscfg, err := tlsutil.ClientConfig(tlsopts)
		if err != nil {
			return nil, err
		}

		opts = append(opts, helm.WithTLS(tlscfg))
	}

	return helm.NewClient(opts...), nil
}

func run(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

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
	case "table":
		table := uitable.New()
		table.AddRow("RELEASE", "CHART", "INSTALLED_VERSION", "LATEST_VERSION", "STATUS")
		for _, versionInfo := range result {
			table.AddRow(versionInfo.ReleaseName, versionInfo.ChartName, versionInfo.InstalledVersion, versionInfo.LatestVersion, versionInfo.Status)
		}
		fmt.Println(table)

	default:
		return fmt.Errorf("invalid formatter: %s", outputFormat)
	}

	return nil
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
