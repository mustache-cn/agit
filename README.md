# agit

![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)

![logo.png](img/logo.png)

**Agit - Git Source Code Batch Acquisition Assistant Tool**

**Agit** is an Git Source Code Batch Acquisition Assistant,designed to facilitate batch retrieval of Git repository source code, aiming to streamline the process of obtaining large-scale code in practical work.

## Core Functions

1. **Batch Acquisition of Source Code**: Users can specify a list of Git repository addresses in the configuration file and automatically clone these repositories to a designated local directory by executing a single command, `agit`.

2. **GitLab Support**: Agit comes with built-in support for the GitLab platform, enabling users to automatically retrieve all project source codes they have access to after authentication. Additionally, it supports configuring a list of groups to fetch, enabling more fine-grained source code management.

3. **Exclusion Feature**: To cater to different user needs, Agit offers an exclusion feature. Users can configure a list of groups and repositories to exclude, ensuring only the desired source code is retrieved, avoiding unnecessary resource wastage.

## Usage

1. **Installation**: Download the run file and configure the system environment variables, or execute the agit command directly in the file directory.

2. **Configuration**: Create a configuration file with the default name config.yml and place it in the root directory where the source code is stored. Multiple configuration files are supported and the parameter is -c config2.yml.

```yml
url: https://git.xx.com
token:
path:        # If it is empty, it is the current directory
groups:
    - group1
    - group2
groupIgnore:
    - group3
    - group4/framework
repos:
    - git@git.xxx.com:group/repo1.git
repoIgnore:
    - git@git.xxx.com:group/repo2.git
```

3. **Executing Commands**: After configuring, users simply need to execute the `agit` command in the command line to start the batch acquisition process of source code.

```shell
agit

# OR

agit -c config2.yml
```

## Advantages and Features

1. **Efficiency** :Agit supports batch cloning of multiple repositories, significantly improving the efficiency of source code acquisition.

2. **Flexibility**: Agit supports custom configurations, allowing users to add or remove repository addresses, configure groups to fetch, and exclude specific groups and repositories.

3. **Ease of Use**: Agit provides a concise and intuitive command-line interface, eliminating the need for users to write complex scripts or commands, making it effortless to complete the task of batch acquiring source code.

4. **Cross-Platform Compatibility**: Agit supports multiple operating systems, including macOS, Linux, and Windows, enabling seamless usage across different platforms without worrying about compatibility issues.

## Applicable Scenarios

Agit is suitable for scenarios where multiple Git repository source codes need to be batch acquired, such as team project collaboration, source code backup, and code audits. Especially when it comes to batch acquiring project source codes from platforms like GitLab, Agit greatly simplifies the operation process and improves work efficiency.

## Open Source and Contributions

Agit is an open-source project, and we welcome any developers with a need for Git source code batch acquisition to join our ranks and jointly improve and optimize this tool. We encourage users to submit bug reports, feature suggestions, and code contributions to jointly drive the development of Agit.


## License

RestfulFinder is under the Apache 2.0 license. See the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0) file for details..