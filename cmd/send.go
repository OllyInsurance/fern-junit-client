package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guidewire-oss/fern-junit-client/pkg/client"
)

var (
	fernUrl     string
	projectId   string
	filePattern string
	tags        string
	branch      string
	commitSha   string
	environment string
	buildURL    string
	metadata    []string
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send JUnit test reports to Fern",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		b, c, e := resolveProvenance(branch, commitSha, environment)
		md, err := resolveMetadata(metadata)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
		if err := client.SendReports(client.SendOptions{
			FernURL: fernUrl, ProjectID: projectId, FilePattern: filePattern, Tags: tags,
			Branch: b, CommitSha: c, Environment: e,
			BuildURL: resolveBuildURL(buildURL), Metadata: md, Verbose: verbose,
		}); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
	},
}

// resolveProvenance fills in git provenance from common CI env vars when the
// corresponding flag was not provided (an explicit flag value always wins).
func resolveProvenance(branch, commit, environment string) (string, string, string) {
	// GITHUB_HEAD_REF is the PR source branch (set only on pull_request events) and
	// takes precedence over GITHUB_REF_NAME, which on PR builds is the synthetic
	// refs/pull/<n>/merge ref rather than a real branch name.
	return firstNonEmpty(branch, os.Getenv("GITHUB_HEAD_REF"), os.Getenv("GITHUB_REF_NAME"), os.Getenv("CI_COMMIT_REF_NAME")),
		firstNonEmpty(commit, os.Getenv("GITHUB_SHA"), os.Getenv("CI_COMMIT_SHA")),
		firstNonEmpty(environment, os.Getenv("CI_ENVIRONMENT_NAME"), os.Getenv("FERN_ENVIRONMENT"))
}

// resolveBuildURL is the CI job's URL: the flag, else the GitHub Actions run.
func resolveBuildURL(flag string) string {
	if flag != "" {
		return flag
	}
	srv, repo, id := os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY"), os.Getenv("GITHUB_RUN_ID")
	if srv == "" || repo == "" || id == "" {
		return ""
	}
	return srv + "/" + repo + "/actions/runs/" + id
}

// resolveMetadata is the run's CI provenance from GitHub Actions env vars,
// overlaid with --metadata key=value pairs (a flag wins over the env).
func resolveMetadata(pairs []string) (map[string]string, error) {
	md := map[string]string{}
	for k, env := range map[string]string{
		"ci_run_id": "GITHUB_RUN_ID", "ci_run_attempt": "GITHUB_RUN_ATTEMPT",
		"ci_job": "GITHUB_JOB", "ci_workflow": "GITHUB_WORKFLOW", "ci_event": "GITHUB_EVENT_NAME",
	} {
		if v := os.Getenv(env); v != "" {
			md[k] = v
		}
	}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("--metadata %q: want key=value", p)
		}
		md[k] = v
	}
	return md, nil
}

// firstNonEmpty returns the first non-empty string from the provided candidates.
func firstNonEmpty(candidates ...string) string {
	for _, s := range candidates {
		if s != "" {
			return s
		}
	}
	return ""
}

func init() {
	sendCmd.PersistentFlags().StringVarP(&fernUrl, "fern-url", "u", "", "base URL of the Fern Platform instance to send test reports to (required)")
	sendCmd.PersistentFlags().StringVarP(&projectId, "project-id", "p", "", "Id of the project to associate test reports with (required). You must register the application first in Fern Platform")
	sendCmd.PersistentFlags().StringVarP(&filePattern, "file-pattern", "f", "", "file name pattern of test reports to send to Fern (required)")
	sendCmd.PersistentFlags().StringVarP(&tags, "tags", "t", "", "comma-separated tags to be included on runs")
	sendCmd.PersistentFlags().StringVar(&branch, "branch", "", "git branch name for this run (falls back to $GITHUB_HEAD_REF / $GITHUB_REF_NAME / $CI_COMMIT_REF_NAME)")
	sendCmd.PersistentFlags().StringVar(&commitSha, "commit", "", "git commit SHA for this run (falls back to $GITHUB_SHA / $CI_COMMIT_SHA)")
	sendCmd.PersistentFlags().StringVar(&environment, "environment", "", "environment label, e.g. ci, staging (falls back to $CI_ENVIRONMENT_NAME / $FERN_ENVIRONMENT)")
	sendCmd.PersistentFlags().StringVar(&buildURL, "build-url", "", "URL of the CI job that produced the reports (falls back to the GitHub Actions run URL)")
	sendCmd.PersistentFlags().StringArrayVar(&metadata, "metadata", nil, "run metadata key=value, repeatable (adds to ci_run_id/ci_run_attempt/ci_job/ci_workflow/ci_event from GitHub Actions)")
	if err := sendCmd.MarkPersistentFlagRequired("fern-url"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if err := sendCmd.MarkPersistentFlagRequired("project-id"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if err := sendCmd.MarkPersistentFlagRequired("file-pattern"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	rootCmd.AddCommand(sendCmd)
}
