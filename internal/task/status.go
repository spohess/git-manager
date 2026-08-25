package task

import (
	"fmt"

	"git-manager/internal/git"
	"git-manager/internal/ui"
)

func runStatus(ctx *Context) error {
	repo := ctx.Git

	if err := repo.Fetch(); err != nil {
		ui.Warn("fetch falhou, mostrando status com as referências locais: %v", err)
	}

	info, err := repo.Status()
	if err != nil {
		return fmt.Errorf("não foi possível obter o status: %w", err)
	}

	branchLabel := info.Branch
	if info.Detached {
		branchLabel = "HEAD destacado"
		ui.Warn("%s", branchLabel)
	} else {
		ui.Step("branch atual: %s", branchLabel)
	}

	switch {
	case info.Upstream == "":
		ui.Warn("sem upstream configurado")
	case info.Ahead == 0 && info.Behind == 0:
		ui.Success("em dia com %s", info.Upstream)
	default:
		ui.Warn("%s: %d à frente, %d atrás", info.Upstream, info.Ahead, info.Behind)
	}

	if info.Clean() {
		ui.Success("árvore de trabalho limpa")
	} else {
		ui.Warn("%d arquivo(s) alterado(s): %d staged, %d não staged, %d não rastreado(s), %d em conflito",
			info.Changed(), info.Staged, info.Unstaged, info.Untracked, info.Conflicts)
	}

	mainStatus := statusVsMain(ctx, info)

	ctx.note("%s", statusDetail(info, branchLabel, mainStatus))
	return nil
}

func statusVsMain(ctx *Context, info git.StatusInfo) string {
	repo := ctx.Git
	mainBranch := repo.DefaultBranch()

	if info.Detached || info.Branch == mainBranch || !repo.RemoteBranchExists(mainBranch) {
		return ""
	}

	ahead, behind, err := repo.AheadBehind(repo.Remote+"/"+mainBranch, "HEAD")
	if err != nil {
		ui.Warn("não foi possível comparar com %s: %v", mainBranch, err)
		return ""
	}
	if ahead == 0 && behind == 0 {
		ui.Success("atualizada com %s", mainBranch)
		return fmt.Sprintf("atualizada com %s", mainBranch)
	}
	ui.Warn("%d commit(s) à frente e %d atrás de %s", ahead, behind, mainBranch)
	return fmt.Sprintf("%d à frente / %d atrás de %s", ahead, behind, mainBranch)
}

func statusDetail(info git.StatusInfo, branchLabel, mainStatus string) string {
	state := "limpa"
	if !info.Clean() {
		state = fmt.Sprintf("%d alteração(ões)", info.Changed())
	}

	upstream := "sem upstream"
	if info.Upstream != "" {
		if info.Ahead == 0 && info.Behind == 0 {
			upstream = fmt.Sprintf("em dia com %s", info.Upstream)
		} else {
			upstream = fmt.Sprintf("%s (%d à frente/%d atrás)", info.Upstream, info.Ahead, info.Behind)
		}
	}

	detail := fmt.Sprintf("%s | %s | %s", branchLabel, state, upstream)
	if mainStatus != "" {
		detail += " | " + mainStatus
	}
	return detail
}
