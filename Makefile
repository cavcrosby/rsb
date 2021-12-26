# special makefile variables
.DEFAULT_GOAL := help
.RECIPEPREFIX := >

# recursive variables
SHELL = /usr/bin/sh
TARGET_EXEC = rsb
AGENT_FILE_PATH = ./rsb.agent
BUILD_DIR = build
BUILD_DIR_PATH = ./${BUILD_DIR}
target_exec_path = ${BUILD_DIR_PATH}/${TARGET_EXEC}

# executables
GO = go
JQ = jq
executables = \
	${JQ}\
	${GO}

DEV_GO_TOOLS = \
	github.com/google/addlicense@v1.0.0

# gnu install directory variables
prefix = /usr/local
exec_prefix = ${prefix}
bin_dir = ${exec_prefix}/bin

# targets
HELP = help
INSTALL = install
UNINSTALL = uninstall
INSTALL_TOOLS = install-tools
MKAGENT_FILE = mkagent-file
ADD_LICENSE = add-license
CLEAN = clean

# to be passed in at make runtime
COPYRIGHT_HOLDERS =

define GRAW_AGENT_FILE =
cat << _EOF_
user_agent: ""
client_id: ""
client_secret: ""
username: ""
password: ""
_EOF_
endef
# Use the $(value ...) function if there are other variables in the multi-line
# variable that should be evaluated by the shell and not make! e.g. 
# export GRAW_AGENT_FILE = $(value _GRAW_AGENT_FILE)
export GRAW_AGENT_FILE

# simply expanded variables
src := $(shell find . \( -type f \) \
	-and \( -name '*.go' \) \
	-and \( -not -iregex '.*/vendor.*' \) \
)
_check_executables := $(foreach exec,${executables},$(if $(shell command -v ${exec}),pass,$(error "No ${exec} in PATH")))

.PHONY: ${HELP}
${HELP}:
	# inspired by the makefiles of the Linux kernel and Mercurial
>	@echo 'Common make targets:'
>	@echo '  ${TARGET_EXEC}                - the ${TARGET_EXEC} binary'
>	@echo '  ${INSTALL}            - installs the rsb binary and other needed files'
>	@echo '  ${UNINSTALL}          - uninstalls the rsb binary and other needed files'
>	@echo '  ${MKAGENT_FILE}       - make agent file needed to use Reddit'\''s API'
>	@echo '  ${INSTALL_TOOLS}      - installs optional development tools used for the project'
>	@echo '  ${ADD_LICENSE}        - adds license header to src files'
>	@echo '  ${CLEAN}              - remove files created by other targets'
>	@echo 'Common make configurations (e.g. make [config]=1 [targets]):'
>	@echo '  COPYRIGHT_HOLDERS     - string denoting copyright holder(s)/author(s)'
>	@echo '                          (e.g. "John Smith, Alice Smith" or "John Smith")'

${TARGET_EXEC}: rsb.go
>	${GO} build -o "${target_exec_path}" -mod vendor

.PHONY: ${INSTALL}
${INSTALL}: ${TARGET_EXEC}
>	${SUDO} ${INSTALL} "${target_exec_path}" "${DESTDIR}${bin_dir}"

.PHONY: ${UNINSTALL}
${UNINSTALL}:
>	${SUDO} rm --force "${DESTDIR}${bin_dir}/${TARGET_EXEC}"

.PHONY: ${MKAGENT_FILE}
${MKAGENT_FILE}:
>	eval "$${GRAW_AGENT_FILE}" > "${AGENT_FILE_PATH}"

# DISCUSS(cavcrosby): in other golang projects, go development tools were part of
# 'go.mod' and inside the vendor directory. Reproducing this setup is a pain and
# I'd like investigate what direction this project and others should take.
# One discussion that might be worth coming back to when addressing this topic:
# https://github.com/golang/go/issues/25922
.PHONY: ${INSTALL_TOOLS}
${INSTALL_TOOLS}:
>	${GO} install -mod mod ${DEV_GO_TOOLS}

.PHONY: ${ADD_LICENSE}
${ADD_LICENSE}:
>	@[ -n "${COPYRIGHT_HOLDERS}" ] || { echo "COPYRIGHT_HOLDERS was not passed into make"; exit 1; }
>	${ADDLICENSE} -l mit -c "${COPYRIGHT_HOLDERS}" ${src}

.PHONY: ${CLEAN}
${CLEAN}:
>	${SUDO} rm --recursive --force "${BUILD_DIR}"
