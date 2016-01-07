#! /bin/bash

# if no arg is given, plugin name will be cf-<cwd-base-name>
# if arg starting with "cf-" is given, plugin name is arg
# else plugin name is cf-arg

set -e
pluginName=${PWD##*/}
if [ -n "$1" ] ; then
	pluginName=$1
fi
if [[ $pluginName != cf-* ]] ; then
	pluginName="cf-$pluginName"
fi

go build -o $pluginName
cf uninstall-plugin $pluginName || echo "Ignoring..."
cf install-plugin $pluginName
