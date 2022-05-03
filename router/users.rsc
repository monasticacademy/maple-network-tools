# For some reason this does not get exported with /export

# First run: scp ~/.ssh/id_rsa.pub admin@microtik.maple.cml.me:
/user add name=koshin group=full
/user ssh-keys import public-key-file=id_rsa.pub user=koshin

/user add name=traffic-monitor group=read
/user set traffic-monitor password=$(cat secrets/traffic-monitor-password)
