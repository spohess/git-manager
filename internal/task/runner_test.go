package task

import (
	"strings"
	"testing"

	"git-manager/internal/config"
)

func TestResolveMajorBranch(t *testing.T) {
	backend := config.Project{Name: "backend", MajorBranch: "develop"}
	admin := config.Project{Name: "admin", MajorBranch: "develop"}
	client := config.Project{Name: "client", MajorBranch: "main"}
	semMajor := config.Project{Name: "site"}

	cases := []struct {
		name     string
		flag     string
		projects []config.Project
		want     string
		wantErr  string
	}{
		{name: "flag prevalece", flag: "release", projects: []config.Project{backend, client}, want: "release"},
		{name: "flag com espaços", flag: "  release ", projects: []config.Project{semMajor}, want: "release"},
		{name: "major-branch único", projects: []config.Project{backend, admin}, want: "develop"},
		{name: "major-branch diferentes", projects: []config.Project{backend, client}, wantErr: "major-branch diferentes"},
		{name: "sem major-branch", projects: []config.Project{backend, semMajor}, wantErr: `"site" não possui "major-branch"`},
	}
	for _, tc := range cases {
		got, err := resolveMajorBranch("target", tc.flag, tc.projects)
		if tc.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("%s: esperado erro contendo %q, obtido %v", tc.name, tc.wantErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: esperado %q, obtido %q", tc.name, tc.want, got)
		}
	}
}

func TestRunExigeTargetSemMajorBranch(t *testing.T) {
	cases := []string{"update", "new", "checkout"}
	for _, name := range cases {
		cfg := &config.Config{Projects: []config.Project{{Name: "site", Provider: "github", Path: t.TempDir(), Main: true}}}
		err := Run(cfg, Options{Name: name, Branch: "feature/x"})
		if err == nil || !strings.Contains(err.Error(), "--source") {
			t.Errorf("%s: esperado erro exigindo --source, obtido %v", name, err)
		}
	}
}
