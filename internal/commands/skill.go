package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/skills"
)

// NewSkillCmd vuelca el skill embebido. Es como un agente aprende a usar este
// binario, sin depender de un repositorio de skills que puede estar desfasado.
func NewSkillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Vuelca la documentación para agentes de esta versión del CLI",
		Args:  NoPositionalArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), skills.SkillMD)
			return nil
		},
	}
}
