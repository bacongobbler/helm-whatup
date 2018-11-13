package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/repo"
)

const globalUsage = `
Check to see if there is an updated version available for installed charts.
`

var outputFormat string

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

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "plain", "Output format")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	host, exists := os.LookupEnv("TILLER_HOST")
	if !exists {
		host = "127.0.0.1:44134"
	}
	client := helm.NewClient(helm.Host(host))

	releases, err := fetchReleases(client)
	if err != nil {
		return err
	}

	repositories, err := fetchIndices(client)
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		fmt.Println("No releases found. All up to date!")
		return nil
	}

	if len(repositories) == 0 {
		fmt.Println("No repositories found. Did you run `helm repo update`?")
		return nil
	}

	var result []ChartVersionInfo

	for _, release := range releases {
		for _, idx := range repositories {
			if idx.Has(release.Chart.Metadata.Name, release.Chart.Metadata.Version) {
				// fetch latest release
				chartVer, err := idx.Get(release.Chart.Metadata.Name, "")
				if err != nil {
					chartVer, err = idx.Get(release.Chart.Metadata.Name, ">0.0.0-0")
					if err != nil {
						fmt.Println(err)
						continue
					}
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
	return res.Releases, nil
}

func fetchIndices(client *helm.Client) ([]*repo.IndexFile, error) {
	indices := []*repo.IndexFile{}
	rfp, exists := os.LookupEnv("HELM_PATH_REPOSITORY_FILE")
	if !exists {
		usr, err := user.Current()
		if err == nil {
			rfp = filepath.FromSlash(usr.HomeDir + "/.helm/repository/repositories.yaml")
		}
	}
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
