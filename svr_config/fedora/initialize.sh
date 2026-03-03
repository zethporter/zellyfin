# Create Necessary Directories
mkdir -p /opt/jellyfin/{config,cache}
mkdir -p /srv/media

# Give Permissions for Drives
sudo chown -R 1000:1000 /opt/jellyfin/{config,cache}

cp jellyfin.container /etc/containers/systemd

sudo systemctl daemon-reload
sudo systemctl enable --now jellyfin.container

## Probably need to add some more nginx stuff here.
