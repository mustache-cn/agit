package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	Url         string   `yaml:"url"`
	Token       string   `yaml:"token"`
	Path        string   `yaml:"path"`
	Groups      []string `yaml:"groups,omitempty"`
	Repos       []string `yaml:"repos,omitempty"`
	RepoIgnore  []string `yaml:"repoIgnore,omitempty"`
	GroupIgnore []string `yaml:"groupIgnore,omitempty"`
}

type Repo struct {
	Url       string
	Domain    string
	GroupPath string
	RepoPath  string
}

var (
	version         = "1.0.0"
	isOutputCommand = false
	initSuccess     = false
	client          *gitlab.Client
	config          Config
)

func init() {
	initConfig()
	initClient()
}

func initConfig() {
	c := flag.String("c", "", "config file name")
	v := flag.Bool("version", false, "Prints the version number")
	flag.Parse()
	if *v {
		isOutputCommand = true
		fmt.Println("Version:", version)
		return
	}
	configFile := "config.yml"
	if len(strings.TrimSpace(*c)) > 0 {
		configFile = *c
	}

	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		color.Red("Configuration initialized failed:%v\n", err)
		return
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		color.Red("Configuration initialized failed:%v\n", err)
		return
	} else {
		fmt.Printf("Configuration initialized successfully: %s\n", configFile)
	}
	if config.Groups == nil {
		config.Groups = []string{}
	}
	if config.Repos == nil {
		config.Repos = []string{}
	}
	if config.RepoIgnore == nil {
		config.RepoIgnore = []string{}
	}
	if config.GroupIgnore == nil {
		config.GroupIgnore = []string{}
	}
	if len(strings.TrimSpace(config.Path)) == 0 {
		config.Path = GetCurrentPath()
	}
	initSuccess = true
}

func initClient() {
	if len(strings.TrimSpace(config.Url)) == 0 || len(strings.TrimSpace(config.Token)) == 0 {
		return
	}
	var err error
	client, err = gitlab.NewClient(config.Token, gitlab.WithBaseURL(config.Url))
	if err != nil {
		color.Red("Client initialized failed:%v\n", err)
		return
	} else {
		fmt.Printf("Client initialization succeeded: %s\n", config.Url)
	}
	initSuccess = true
}

func main() {
	if isOutputCommand || !initSuccess {
		return
	}

	for _, repoUrl := range config.Repos {
		errFetch := FetchRepo(repoUrl)
		if errFetch != nil {
			return
		}
	}

	if client != nil {
		groups, _ := GetAllGroup()
		for _, group := range groups {
			FetchGroup(group)
		}
	}
	color.HiGreen("******** All executed！********")
}

func FetchGroup(group *gitlab.Group) {
	color.Green("******Group: %s Start fetching******\n", group.FullPath)
	projects, _ := GetProjectsByGroup(group.FullPath)
	if len(projects) == 0 {
		color.Red("******Group: %s Not exist skip******\n", group.FullPath)
	}
	for _, project := range projects {
		err := FetchProject(project)
		if err != nil {
			continue
		}
	}
	color.Green("******Group: %s Fetch complete******\n\n", group.FullPath)
}

func FetchProject(project *gitlab.Project) error {
	color.Green("******Project: %s Start fetching******\n", project.Path)
	workPath := filepath.Join(GetWorkPath(), project.PathWithNamespace)
	if IsGitRepo(workPath) {
		existsBranch, branchName, err := ExistsBranch(workPath)
		if err != nil {
			color.Red("%s An error occurred while checking if the local branch exists on the server.", project.Path)
			return err
		}
		if !existsBranch {
			color.Red("%s branch %s not exist on the server, it will be switched to master.", project.Path, branchName)
			err := GitSwitchBranch(workPath, "master")
			if err != nil {
				color.Red("%s An error occurred while switching the branch to master.", project.Path)
				return err
			}
		}
		color.Blue("%s repository already exists. Pulling the latest changes.\n", project.PathWithNamespace)
		GitPull(workPath)
	} else {
		color.Green("%s repository does not exist. Cloning the repository from the remote server.\n", project.PathWithNamespace)
		GitClone(project.SSHURLToRepo, workPath)
	}
	color.Green("******Project: %s Fetch complete******\n", project.Path)
	return nil
}

func GetAllGroup() ([]*gitlab.Group, error) {
	groups, err, done := GetConfigGroups()
	if done {
		return groups, err
	}

	alv := gitlab.AccessLevelValue(30)
	lgo := &gitlab.ListGroupsOptions{ListOptions: gitlab.ListOptions{Page: 1, PerPage: 100}, MinAccessLevel: &alv}
	var gro []*gitlab.Group
	for {
		g, _, err := client.Groups.ListGroups(lgo)
		if err != nil {
			color.Red("list groups failed:%v\n", err)
			return nil, err
		}
		gro = append(gro, g...)

		for _, gs := range g {
			gss, err := GetSubGroups(gs.ID)
			if err != nil {
				color.Red("list sub group failed:%s %s\n", gs.FullPath, err)
			}
			gro = append(gro, gss...)
		}

		if len(g) < 50 {
			break
		}
		lgo.ListOptions.Page++
	}
	gro = filterGroups(gro)
	return gro, nil
}

func filterGroups(gro []*gitlab.Group) []*gitlab.Group {
	if len(config.GroupIgnore) > 0 {
		gro = filterGroup(gro, func(r *gitlab.Group) bool {
			return !contains(config.GroupIgnore, r.FullPath)
		})

	}
	return gro
}

func GetConfigGroups() ([]*gitlab.Group, error, bool) {
	values := config.Groups
	if len(values) > 0 {
		groups := make([]*gitlab.Group, len(values))
		for i, value := range values {
			group := &gitlab.Group{
				Name:     value,
				FullPath: value,
				Path:     value,
			}
			groups[i] = group
			subGroups, err := GetSubGroups(value)
			if err != nil {
				color.Red("list sub group failed:%s %s\n", value, err)
			}
			groups = append(groups, subGroups...)
		}
		return groups, nil, true
	}
	return nil, nil, false
}

func filterGroup(repos []*gitlab.Group, f func(r *gitlab.Group) bool) []*gitlab.Group {
	result := make([]*gitlab.Group, 0, len(repos))
	for _, r := range repos {
		if f(r) {
			result = append(result, r)
		}
	}
	return result
}

func GetSubGroups(gid interface{}) ([]*gitlab.Group, error) {
	alv := gitlab.AccessLevelValue(30)
	lgo := &gitlab.ListSubGroupsOptions{ListOptions: gitlab.ListOptions{Page: 1, PerPage: 100}, MinAccessLevel: &alv}
	var gro []*gitlab.Group
	for {
		g, _, err := client.Groups.ListSubGroups(gid, lgo)
		if err != nil {
			color.Red("list groups failed:%v\n", err)
			return nil, err
		}
		g = filterGroups(g)
		gro = append(gro, g...)

		for _, gs := range g {
			gss, err := GetSubGroups(gs.ID)
			if err != nil {
				return nil, err
			}
			gro = append(gro, gss...)
		}

		if len(g) < 50 {
			break
		}
		lgo.ListOptions.Page++
	}
	return gro, nil
}

func GetProjectsByGroup(gid interface{}) ([]*gitlab.Project, error) {
	alv := gitlab.AccessLevelValue(30)
	lgo := &gitlab.ListGroupProjectsOptions{ListOptions: gitlab.ListOptions{Page: 1, PerPage: 100}, MinAccessLevel: &alv}
	var pro []*gitlab.Project
	for {
		g, _, err := client.Groups.ListGroupProjects(gid, lgo)
		if err != nil {
			color.Red("list projects failed:%v\n", err)
			return nil, err
		}
		pro = append(pro, g...)
		if len(g) < 50 {
			break
		}
		lgo.ListOptions.Page++
	}
	if len(config.RepoIgnore) > 0 {
		pro = filterProject(pro, func(r *gitlab.Project) bool {
			return !contains(config.RepoIgnore, r.SSHURLToRepo)
		})

	}
	return pro, nil
}

func filterProject(repos []*gitlab.Project, f func(r *gitlab.Project) bool) []*gitlab.Project {
	result := make([]*gitlab.Project, 0, len(repos))
	for _, r := range repos {
		if f(r) {
			result = append(result, r)
		}
	}
	return result
}

func ExistsBranch(dirPath string) (bool, string, error) {
	currentBranch, err := GetCurrentBranch(dirPath)
	if err != nil {
		return false, "", err
	}

	branch, err := GetRemoteBranch(currentBranch, dirPath)
	if err != nil {
		return false, "", err
	}
	if strings.HasSuffix(branch, "/"+currentBranch) {
		return true, currentBranch, nil
	}
	return false, currentBranch, nil
}

func GetRemoteBranch(currentBranch, dirPath string) (string, error) {
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", currentBranch)
	cmd.Dir = dirPath
	return GitCommandRunOutput(cmd)
}

func GetCurrentBranch(dirPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dirPath
	return GitCommandRunOutput(cmd)
}

func GitClone(url, dirPath string) {
	cmd := exec.Command("git", "clone", "--progress", url, dirPath)
	err := GitCommandRun(cmd)
	if err != nil {
		color.Red("%s %s failed！", dirPath, "clone")
	}
}

func GitPull(dirPath string) {
	cmd := exec.Command("git", "pull")
	cmd.Dir = dirPath
	err := GitCommandRun(cmd)
	if err != nil {
		color.Red("%s %s failed！", dirPath, "pull")
	}
}

func GitSwitchBranch(dirPath, branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = dirPath
	err := GitCommandRun(cmd)
	if err != nil {
		color.Red("%s %s %s failed！", dirPath, "checkout", branchName)
		return err
	}
	return nil
}

func GitCommandRun(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func GitCommandRunOutput(cmd *exec.Cmd) (string, error) {
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func GetWorkPath() string {
	dir := config.Path
	if len(strings.TrimSpace(dir)) == 0 {
		return GetCurrentPath()
	}
	return dir
}

func GetCurrentPath() string {
	// 获取当前工作目录
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("dir:" + dir)
	return dir
}

func IsGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

func FetchRepo(url string) error {
	repo, err := GetGitRepo(url)
	if err != nil {
		return err
	}
	color.Green("******Project: %s Start pulling******\n", repo.Url)
	workPath := filepath.Join(GetWorkPath(), repo.GroupPath, repo.RepoPath)
	if IsGitRepo(workPath) {
		existsBranch, branchName, err := ExistsBranch(workPath)
		if err != nil {
			color.Red("%s An error occurred while checking if the local branch exists on the server.", workPath)
			return err
		}
		if !existsBranch {
			color.Red("%s branch %s not exist on the server, it will be switched to master.", workPath, branchName)
			err := GitSwitchBranch(workPath, "master")
			if err != nil {
				color.Red("%s An error occurred while switching the branch to master.", workPath)
				return err
			}
		}
		color.Blue("%s repository already exists. Pulling the latest changes.\n", workPath)
		GitPull(workPath)
	} else {
		color.Green("%s repository does not exist. Cloning the repository from the remote server.\n", workPath)
		GitClone(repo.Url, workPath)
	}
	return nil
}

func GetGitRepo(url string) (*Repo, error) {
	if !IsGitURL(url) {
		color.Red("%s is not git url!", url)
		return nil, errors.New("is not git url")
	}

	rex := `^(https?:\/\/.*?)(/.*)/(.*)\.git$`
	if strings.HasPrefix(strings.ToLower(url), "git@") {
		rex = `^git@(.*):(.*)/(.*)\.git$`
	}
	re := regexp.MustCompile(rex)
	matches := re.FindStringSubmatch(url)

	repoName := matches[3]
	groupName := matches[2]
	domain := matches[1]
	return &Repo{Domain: domain, GroupPath: groupName, RepoPath: repoName, Url: url}, nil
}

func IsGitURL(url string) bool {
	gitUrl := strings.ToLower(url)
	if !strings.HasPrefix(gitUrl, "git@") &&
		!strings.HasPrefix(gitUrl, "http://") &&
		!strings.HasPrefix(gitUrl, "https://") &&
		!strings.HasSuffix(gitUrl, ".git") {
		return false
	}
	return true
}

func contains(s []string, e string) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
