

# create a read-only user account for the monitor tool
ssh admin@router.lan /user add name=traffic-monitor group=read password=<password>

# create the bigquery table (schema should match usage.proto)
mk -t maple.router_usage begin:string,duration:integer,host:string,bytes:integer,packets:integer
