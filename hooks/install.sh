#! /bin/bash

DOT_GIT="$(dirname $0)/../.git"

rm -f "$DOT_GIT/hooks/pre-commit"
ln -s "../../hooks/pre-commit" "$DOT_GIT/hooks/pre-commit"