# 清理非必要的 Git 忽略项

## Goal

移除过宽或不必要的忽略规则，保留运行时敏感配置、构建产物和本地缓存的必要忽略。

## Requirements

- Keep ignores for `config.json`, runtime logs, PID files, and the local build
  binary.
- Remove repository configuration paths from `.gitignore` so Trellis, agent
  instructions, and Git attributes can be versioned.
- Do not change application code or delete any existing files.

## Acceptance Criteria

- [x] `.gitignore` contains only necessary local runtime/build ignores.
- [x] `git check-ignore` confirms the intended runtime files remain ignored and
  project workflow files are no longer ignored.
- [x] Existing tests/build remain green.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
