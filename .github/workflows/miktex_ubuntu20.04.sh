#! /bin/bash

# Ensure the script doesnt hang waiting for user input
export DEBIAN_FRONTEND=noninteractive

# Add packages needed for adding a new repo, then add its key and the repo itself
echo "Installing packages needed for adding repositories ..."
apt update && apt install -y tzdata gnupg ca-certificates
apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys D6BC243565B2087BC3F897C9277A7293F59E4889
echo "deb http://miktex.org/download/ubuntu focal universe" >> /etc/apt/sources.list.d/miktex.list

# Refresh repos and install miktex
echo "Installing MiKTeX ..."
apt update && apt install -y --no-install-recommends miktex

echo "Running post-installation process for MiKTeX ..."
# Miktex needs to run some post-installation scripts
miktexsetup --shared=yes finish
initexmf --admin --set-config-value [MPM]AutoInstall=1
mpm --find-updates

echo "Finished installing MiKTeX!"

