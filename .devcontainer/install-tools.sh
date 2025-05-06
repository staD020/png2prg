#!/bin/bash
# filepath: .devcontainer/install-tools.sh

# Install spot13
cd /tmp
wget -q https://sourceforge.net/projects/spot13/files/latest/download -O spot13.tar.gz
tar -xzf spot13.tar.gz
cd spot13*
make
cp spot13 /usr/local/bin/
cd /tmp
rm -rf spot13*

# Install DALI (if available)
cd /tmp
git clone https://github.com/cadaver/dali.git
cd dali
make
cp dali /usr/local/bin/
cd /tmp
rm -rf dali

# Install d64 utility
cd /tmp
wget -q https://singularcrew.hu/idoproject/d64/d64.zip -O d64.zip || echo "Warning: Could not download d64, you may need to install it manually"
if [ -f d64.zip ]; then
  unzip d64.zip
  chmod +x d64
  cp d64 /usr/local/bin/
  rm d64.zip
fi

# Create directory for tools
mkdir -p /opt/c64tools

# Ensure all binaries are in PATH
echo 'export PATH=$PATH:/opt/c64tools' >> /etc/bash.bashrc