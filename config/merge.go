package config

import (
	"log"
	"strings"

	"github.com/minkolazer/gp/lib"

	"github.com/imdario/mergo"
	"github.com/mink0/exec-cmd"

	"github.com/pkg/errors"
)

// EnvExec is wrapper on Env with Local and Remote commands initialized
type EnvExec struct {
	Env   Env
	Local execmd.Cmd
	// only one remote command
	Remote execmd.ClusterSSHCmd
}

// InitEnv read config files, resolve parents,
// return initialized Env for envName
// TODO
// 	- templating
// 	-	refactor:
//		- make slice of targets,parents,defaults
//		- merge them in one action (func mergo.Map)
func InitEnv(envName string, targetNames []string) (targets []EnvExec, err error) {
	var (
		config map[string]Env // unitialized envs from config files
	)

	log.Printf(`init environment for %s:%s using config: %s...`, envName, targetNames, ConfigPath)

	// read config
	cfg := getConfig()

	// check config contains requested envName
	if _, ok := cfg[envName]; !ok {
		err = errors.Errorf("unknown environment: %s", envName)
		return
	}

	env := EnvExec{
		Env: cfg[envName],
	}

	// merge `General` field into `Local` and `Remote` fields
	if err = mergo.Merge(&env.Env.Remote, env.Env.Defaults); err != nil {
		return
	}
	if err = mergo.Merge(&env.Env.Local, env.Env.Defaults); err != nil {
		return
	}

	// get list of env parents
	envParents, err := getParents(cfg, envName)
	if err != nil {
		err = errors.Wrapf(err, `can't resolve %s environment parents: %v`, envName, envParents)
		return
	}

	if len(envParents) > 0 {
		log.Printf(`found "%s" parents: %v`, envName, envParents)

		// merge parents fields
		for _, e := range envParents {
			if err = mergo.Merge(&env.Env.Defaults, config[e].Defaults); err != nil {
				return
			}
			if err = mergo.Merge(&env.Env.Local, config[e].Local); err != nil {
				return
			}
			if err = mergo.Merge(&env.Env.Remote, config[e].Remote); err != nil {
				return
			}
		}

		// when parents merged, new data may appear in `Defaults` fields,
		// so merge `Defaults` fields into `Local` and `Remote`
		if err = mergo.Merge(&env.Env.Remote, env.Env.Defaults); err != nil {
			return
		}
		if err = mergo.Merge(&env.Env.Local, env.Env.Defaults); err != nil {
			return
		}
	}

	// init exec wrappers
	env.Local = *execmd.NewCmd()
	env.Remote = *execmd.NewClusterSSHCmd(env.Env.Remote.Hosts)

	if len(env.Env.Targets) == 0 || len(targetNames) == 0 {
		targets = append(targets, env)
		return
	}

	/*
		targets initialization

		best effort: target argument maybe not a target,
		throw error only if we have a list of the targets
	*/

	for _, tname := range targetNames {
		tEnv := EnvExec{}
		if err = mergo.Merge(&tEnv, env); err != nil {
			return
		}

		t, ok := tEnv.Env.Targets[tname]
		if !ok {
			err = errors.Errorf(`unknown target "%s" for "%s" environment`, tname, envName)
			return
		}

		// merge target environment
		if err = mergo.Merge(&t.Local, t.Defaults); err != nil {
			return
		}
		if err = mergo.Merge(&t.Remote, t.Defaults); err != nil {
			return
		}

		// merge target environment and override fields which are not default
		if err = mergo.Merge(&tEnv.Env.Defaults, t.Defaults, mergo.WithOverride); err != nil {
			return
		}
		if err = mergo.Merge(&tEnv.Env.Local, t.Local, mergo.WithOverride); err != nil {
			return
		}
		if err = mergo.Merge(&tEnv.Env.Remote, t.Remote, mergo.WithOverride); err != nil {
			return
		}

		// re-init exec wrappers
		tEnv.Remote = *execmd.NewClusterSSHCmd(tEnv.Env.Remote.Hosts)
		for _, s := range tEnv.Remote.SSHCmds {
			s.Cmd.PrefixStdout = strings.TrimSpace(s.Cmd.PrefixStdout) + lib.Color("|"+tname+" ")
			s.Cmd.PrefixStderr += strings.Split(s.Cmd.PrefixStderr, "@")[0] +
				lib.Color("|"+tname+" ") + lib.ColorErr("@err")
			s.Cmd.PrefixCmd += "(" + tname + ") "
		}

		targets = append(targets, tEnv)
	}

	return
}

// GetEnvs reads config and produce list of envs for auto completion
func GetEnvs() (envs []string) {
	config := getConfig()

	for name, env := range config {
		if !env.Hidden && name != "default" {
			envs = append(envs, name)
		}
	}

	return
}

// GetTargets returns list of target names for the env
func GetTargets(env string) (targets []string, err error) {
	config := getConfig()

	if _, ok := config[env]; !ok {
		err = errors.Errorf("unknown environment %s", env)
		return
	}

	for k := range config[env].Targets {
		targets = append(targets, k)
	}

	return
}

// recursive parents search
func getParents(envs map[string]Env, envName string) (parents []string, err error) {
	var (
		walker func(envs map[string]Env, envName string) []string
	)

	walker = func(envs map[string]Env, envName string) []string {
		if lib.ArrayContains(parents, envName) != -1 {
			err = errors.New("circular parent reference:\n" + strings.Join(append(parents, envName), " > "))
			return parents
		}

		if _, ok := envs[envName]; !ok {
			err = errors.Errorf(`unknown parent: "%s"`, envName)
			return parents
		}

		parents = append(parents, envName)
		// prepend to reverse list of parents
		// parents = append([]string{envName}, parents...)

		if envs[envName].Parent != "" {
			envName = envs[envName].Parent
			return walker(envs, envName)
		}

		// all envs should inherit `default`` property if defined
		if _, ok := envs["default"]; ok {
			parents = append(parents, "default")
		}

		return parents
	}

	return walker(envs, envName)[1:], err
}