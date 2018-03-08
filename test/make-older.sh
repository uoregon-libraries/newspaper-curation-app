iam=$(whoami)
sudo chown -R $iam .
find fakemount/ -exec touch -d "4 days ago" {} \;
