#!/bin/bash

# Fail on error
set -e

# Function to print usage
print_usage() {
	echo "Usage: $0 <command> [version] [requirements]"
	echo "Commands:"
	echo "  setup               - Setup pyenv"
	echo "  install <version>   - Install Python version (default: latest)"
	echo "  uninstall <version> - Uninstall Python version"
	echo "  list                - List installed Python versions"
}

# Function to install Python dependencies
install_dependencies() {
	# Update package lists
	apt-get update -y
	
	apt install -y \
		make \
		build-essential \
		libssl-dev \
		zlib1g-dev \
		libxml2-dev \
		libxslt-dev \
		libffi-dev \
		libbz2-dev \
		libreadline-dev \
		libsqlite3-dev \
		zlib1g-dev \
		xz-utils \
		liblzma-dev
}

# Function to setup pyenv
setup_pyenv() {
	# Install pyenv if not already installed
	if [ ! -d "$HOME/.pyenv" ]; then
		curl https://pyenv.run | bash
	fi
		
	if [ ! -f "$HOME/.pyenv-env.sh" ]; then
		# Create a file in $HOME/.pyenv-env.sh
		cat > $HOME/.pyenv-env.sh << 'EOF'
export PYENV_ROOT="$HOME/.pyenv"
[[ -d $PYENV_ROOT/bin ]] && export PATH="$PYENV_ROOT/bin:$PATH"
eval "$(pyenv init -)"
eval "$(pyenv virtualenv-init -)"
EOF
		chmod +x $HOME/.pyenv-env.sh
	fi
	
	source $HOME/.pyenv-env.sh
}

# Function to verify Python installation
verify_python() {
	local version=$1
	python_version=$(python -V)
	if [[ ! $python_version =~ "Python ${version}" ]]; then
		echo "ERROR: python version does not match. expect \"Python ${version}\", but actual is \"${python_version}\""
		return 1
	fi
	
	if ! command -v pip &> /dev/null; then
		echo "ERROR: pip is not installed"
		return 1
	fi
	return 0
}

# Function to create symlinks
create_symlinks() {
	ln -sf $(pyenv which python) /usr/local/bin/python
	ln -sf $(pyenv which python3) /usr/local/bin/python3
	ln -sf $(pyenv which pip) /usr/local/bin/pip
}

# Function to cleanup
cleanup() {
	pip cache purge
	rm -rf ~/.cache/pip/*
	apt-get remove -y make build-essential
	apt-get autoremove -y
}

# Function to handle requirements
handle_requirements() {
	local requirements_content="$1"
	if [ -n "$requirements_content" ]; then
		REQUIREMENTS_FILE="/tmp/requirements_$(date +%s)_$RANDOM.txt"
		echo "$requirements_content" > "$REQUIREMENTS_FILE"
		pip install -r "$REQUIREMENTS_FILE"
		rm "$REQUIREMENTS_FILE"
	else
		# Fallback to default requirements file if it exists
		if [ -f "/app/install/python/requirements.txt" ]; then
			pip install -r /app/install/python/requirements.txt
		fi
	fi
}

# Main logic
command="${1:-}"
version="${2:-}"
requirements="${3:-}"

case $command in
	"setup")
		setup_pyenv
		;;
	"install")
		if [ -z "$version" ]; then
			echo "Please specify a version to install"
			exit 1
		fi
		setup_pyenv
		# Check if version is already installed
		if pyenv versions | grep -q $version; then
			echo "Python $version is already installed. Switching to it..."
			pyenv global $version
		else
			install_dependencies
			pyenv install $version
			pyenv global $version
		fi
		verify_python $version
		handle_requirements "$requirements"
		create_symlinks
		cleanup
		;;
	
	"uninstall")
		if [ -z "$version" ]; then
			echo "Please specify a version to uninstall"
			exit 1
		fi
		setup_pyenv
		pyenv uninstall -f $version
		;;
	
	"list")
		setup_pyenv
		pyenv install --list | awk '/^  [23]/ {print $1}' | grep -v "[a-zA-Z]" | tac
		;;

	*)
		print_usage
		exit 1
		;;
esac