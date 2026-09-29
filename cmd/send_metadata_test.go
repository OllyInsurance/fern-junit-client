package cmd

import "testing"

func TestResolveMetadataFromEnvAndFlags(t *testing.T) {
	t.Setenv("GITHUB_RUN_ID", "36539680188")
	t.Setenv("GITHUB_RUN_ATTEMPT", "2")
	t.Setenv("GITHUB_JOB", "replica-lane")
	md, err := resolveMetadata([]string{"lane=e2e", "ci_job=override"})
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"ci_run_id": "36539680188", "ci_run_attempt": "2", "ci_job": "override", "lane": "e2e"} {
		if md[k] != want {
			t.Errorf("%s = %q, want %q", k, md[k], want)
		}
	}
	if _, err := resolveMetadata([]string{"novalue"}); err == nil {
		t.Error("a pair without = must be rejected")
	}
}

func TestResolveBuildURL(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "OllyInsurance/olly")
	t.Setenv("GITHUB_RUN_ID", "1")
	if got := resolveBuildURL(""); got != "https://github.com/OllyInsurance/olly/actions/runs/1" {
		t.Errorf("got %q", got)
	}
	if got := resolveBuildURL("https://x"); got != "https://x" {
		t.Errorf("flag must win, got %q", got)
	}
}
