package task

import (
	"strings"
	"testing"

	"git-manager/internal/config"
)

func TestResolveTarget(t *testing.T) {
	backend := config.Project{Name: "backend", BranchTarget: "develop"}
	admin := config.Project{Name: "admin", BranchTarget: "develop"}
	client := config.Project{Name: "client", BranchTarget: "main"}
	semTarget := config.Project{Name: "site"}

	cases := []struct {
		name     string
		flag     string
		projects []config.Project
		want     string
		wantErr  string
	}{
		{name: "flag prevalece", flag: "release", projects: []config.Project{backend, client}, want: "release"},
		{name: "flag com espaços", flag: "  release ", projects: []config.Project{semTarget}, want: "release"},
		{name: "branch-target único", projects: []config.Project{backend, admin}, want: "develop"},
		{name: "branch-target diferentes", projects: []config.Project{backend, client}, wantErr: "branch-target diferentes"},
		{name: "sem branch-target", projects: []config.Project{backend, semTarget}, wantErr: `"site" não possui "branch-target"`},
	}
	for _, tc := range cases {
		got, err := resolveTarget(tc.flag, tc.projects)
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
