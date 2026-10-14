package main

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var embeddedVersion string

//go:embed installer/AGENTS.rule.md
var embeddedAgentsRule string

//go:embed skill-template/myautoapi-image-generation/SKILL.md
var embeddedSkill string

//go:embed skill-template/myautoapi-image-generation/agents/openai.yaml
var embeddedAgentConfig string

func version() string {
	return strings.TrimSpace(embeddedVersion)
}
