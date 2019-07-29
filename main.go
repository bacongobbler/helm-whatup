// Copyright (c) 2019 FABMation GmbH
// All Rights Reserved
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/jedib0t/go-pretty/table"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"k8s.io/helm/pkg/helm"
	helmenv "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/tlsutil"
)

var outputFormat string
var devel bool
var logDebug bool
var version = "canary"

var (
	settings helmenv.EnvSettings
)

const (
	statusOutdated = "OUTDATED"
	statusUptodate = "UPTODATE"
)

// ChartVersionInfo contains all relevant Informations about Chart Releases in Tiller/ Helm
type ChartVersionInfo struct {
	ReleaseName      string `json:"releaseName"`      // Helm Release Name
	ChartName        string `json:"chartName"`        // Chart Name of the Release
	InstalledVersion string `json:"installedVersion"` // Installed Chart Version
	LatestVersion    string `json:"latestVersion"`    // Latest available Version of Chart
	Status           string `json:"status"`           // Status of Release: Is Release UpToDate or Outdated
}

func main() {
	cmd := &cobra.Command{
		Use:   "whatup [flags]",
		Short: fmt.Sprintf("check if installed charts are out of date"),
		RunE:  run,
		Version: version,
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format, choose from plain, json, yaml, table")
	cmd.Flags().BoolVarP(&devel, "devel", "d", false, "Whether to include pre-releases or not, defaults to false.")
	cmd.Flags().BoolVarP(&logDebug, "deb", "D", false, "Print Debug Logs, defaults to false.")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
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

	result, err := parseReleases(releases, repositories)
	if err != nil {
		return err
	}

	// output Informations
	err = formatOutput(result)
	if err != nil {
		debug("There was an Error while formatting and printing the Results")
		return err
	}

	return nil
}

func newClient() (*helm.Client, error) {
	/// === Pre-Checks ===
	if settings.TillerHost == "" {
		if os.Getenv("TILLER_HOST") != "" {
			settings.TillerHost = os.Getenv("TILLER_HOST")

		} else if os.Getenv("HELM_HOST") != "" {
			settings.TillerHost = os.Getenv("HELM_HOST")
		}

		if settings.TillerHost == "" {
			return nil, fmt.Errorf("error: Tiller Host not set")
		}
	}

	if settings.TLSCaCertFile == helmenv.DefaultTLSCaCert || settings.TLSCaCertFile == "" {
		settings.TLSCaCertFile = fmt.Sprintf("%s/%s", os.ExpandEnv("$HELM_HOME"), settings.Home.TLSCaCert())
	} else {
		settings.TLSCaCertFile = os.ExpandEnv(settings.TLSCaCertFile)
	}

	if settings.TLSCertFile == helmenv.DefaultTLSCert || settings.TLSCertFile == "" {
		settings.TLSCertFile = fmt.Sprintf("%s/%s", os.ExpandEnv("$HELM_HOME"), settings.Home.TLSCert())
	} else {
		settings.TLSCertFile = os.ExpandEnv(settings.TLSCertFile)
	}

	if settings.TLSKeyFile == helmenv.DefaultTLSKeyFile || settings.TLSKeyFile == "" {
		settings.TLSKeyFile = fmt.Sprintf("%s/%s", os.ExpandEnv("$HELM_HOME"), settings.Home.TLSKey())
	} else {
		settings.TLSKeyFile = os.ExpandEnv(settings.TLSKeyFile)
	}

	if os.Getenv("HELM_TLS_ENABLE") != "" {
		settings.TLSEnable, _ = strconv.ParseBool(os.Getenv("HELM_TLS_ENABLE"))
	}

	if os.Getenv("HELM_TLS_VERIFY") != "" {
		settings.TLSVerify, _ = strconv.ParseBool(os.Getenv("HELM_TLS_VERIFY"))
	}

	options := []helm.Option{helm.Host(settings.TillerHost)}

	debug("Tiller Host: \"%s\", TLS Enabled: \"%t\", TLS Verify: \"%t\"",
		settings.TillerHost, settings.TLSEnable, settings.TLSVerify)

	// check if TLS is enabled
	if settings.TLSEnable || settings.TLSVerify {
		debug("Host=%q, Key=%q, Cert=%q, CA=%q\n", settings.TLSServerName, settings.TLSKeyFile, settings.TLSCertFile, settings.TLSCaCertFile)

		tlsopts := tlsutil.Options{
			ServerName:         settings.TillerHost,
			CaCertFile:         settings.TLSCaCertFile,
			CertFile:           settings.TLSCertFile,
			KeyFile:            settings.TLSKeyFile,
			InsecureSkipVerify: !settings.TLSVerify,
		}

		tlscfg, err := tlsutil.ClientConfig(tlsopts)

		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			return nil, err
		}

		options = append(options, helm.WithTLS(tlscfg))
	}

	return helm.NewClient(options...), nil
}

func formatOutput(result []ChartVersionInfo) error {
	switch outputFormat {
	case "table":
		_table := table.NewWriter()
		_table.SetOutputMirror(os.Stdout)

		_table.AppendHeader(table.Row{"Release Name", "Installed version", "Available version"})

		for _, versionInfo := range result {
			if versionInfo.LatestVersion != versionInfo.InstalledVersion {
				_table.AppendRow(table.Row{versionInfo.ReleaseName, versionInfo.InstalledVersion, versionInfo.LatestVersion})
			}
		}

		// print Table
		_table.Render()

	case "plain":
		for _, versionInfo := range result {
			if versionInfo.LatestVersion != versionInfo.InstalledVersion {
				fmt.Printf("There is an update available for helm_release %s (%s)!\nInstalled version: %s\nAvailable version: %s\n", versionInfo.ReleaseName, versionInfo.ChartName, versionInfo.InstalledVersion, versionInfo.LatestVersion)
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
		return fmt.Errorf("invalid output formatter: '%s'", outputFormat)
	}

	return nil
}

func parseReleases(releases []*release.Release, repositories []*repo.IndexFile) ([]ChartVersionInfo, error) {
	var result []ChartVersionInfo

	for _, helmRelease := range releases {
		for _, idx := range repositories {
			if idx.Has(helmRelease.Chart.Metadata.Name, helmRelease.Chart.Metadata.Version) {
				// fetch latest helm_release
				constraint := ""
				// Include pre-releases
				if devel {
					constraint = ">= *-0"
				}
				chartVer, err := idx.Get(helmRelease.Chart.Metadata.Name, constraint)

				if err != nil {
					return nil, err
				}

				versionStatus := ChartVersionInfo{
					ReleaseName:      helmRelease.Name,
					ChartName:        helmRelease.Chart.Metadata.Name,
					InstalledVersion: helmRelease.Chart.Metadata.Version,
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

	return result, nil
}

func fetchReleases(client *helm.Client) ([]*release.Release, error) {
	res, err := client.ListReleases()

	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.New("no releases found :(")
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

	if len(repofile.Repositories) == 0 {
		return nil, errors.New("no repositories found. run `helm repo update` and re-try")
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

func debug(format string, args ...interface{}) {
	if logDebug {
		format = fmt.Sprintf("[DEBUG] %s\n", format)
		fmt.Printf(format, args...)
	}
}
