# may/11/2022 13:06:04 by RouterOS 6.49.2
# software id = V1CM-1UH3
#
# model = RB4011iGS+
# serial number = FAD40FEC9992
/interface bridge
add admin-mac=DC:2C:6E:66:8C:32 auto-mac=no comment=defconf name=bridge
/interface ethernet switch port
set 0 default-vlan-id=0
set 1 default-vlan-id=0
set 2 default-vlan-id=0
set 3 default-vlan-id=0
set 4 default-vlan-id=0
set 5 default-vlan-id=0
set 6 default-vlan-id=0
set 7 default-vlan-id=0
set 8 default-vlan-id=0
set 9 default-vlan-id=0
set 10 default-vlan-id=0
set 11 default-vlan-id=0
/interface list
add comment=defconf name=WAN
add comment=defconf name=LAN
add comment="koshin 5-3-2022" name=VTEL
/interface wireless security-profiles
set [ find default=yes ] supplicant-identity=MikroTik
/ip pool
add name=default-dhcp ranges=192.168.88.10-192.168.88.254
/ip dhcp-server
add address-pool=default-dhcp disabled=no interface=bridge name=defconf
/system logging action
add disk-file-name=packetlog name=packetlog target=disk
/interface bridge port
add bridge=bridge comment=defconf interface=ether2
add bridge=bridge comment=defconf interface=ether3
add bridge=bridge comment=defconf interface=ether4
add bridge=bridge comment=defconf interface=ether5
add bridge=bridge comment=defconf interface=ether6
add bridge=bridge comment=defconf interface=ether7
add bridge=bridge comment=defconf interface=ether8
add bridge=bridge comment=defconf interface=ether9
add bridge=bridge comment=defconf interface=sfp-sfpplus1
/ip neighbor discovery-settings
set discover-interface-list=LAN
/interface list member
add comment=defconf interface=bridge list=LAN
add comment=defconf interface=ether1 list=WAN
add comment="koshin 5-3-2022" interface=ether10 list=VTEL
/ip accounting
set enabled=yes
/ip accounting web-access
set accessible-via-web=yes
/ip address
add address=192.168.88.1/24 comment=defconf interface=bridge network=\
    192.168.88.0
/ip dhcp-client
add comment=defconf default-route-distance=2 disabled=no interface=ether1
add comment="koshin 5-3-2022" default-route-distance=10 disabled=no \
    interface=ether10 use-peer-dns=no use-peer-ntp=no
/ip dhcp-server lease
add address=192.168.88.250 client-id=1:90:9:d0:0:60:b7 mac-address=\
    90:09:D0:00:60:B7 server=defconf
add address=192.168.88.248 client-id=1:78:d2:94:9b:f3:2b mac-address=\
    78:D2:94:9B:F3:2B server=defconf
add address=192.168.88.247 client-id=1:f0:92:1c:d9:88:a0 mac-address=\
    F0:92:1C:D9:88:A0 server=defconf
add address=192.168.88.239 client-id=1:b0:68:e6:6b:a5:b1 mac-address=\
    B0:68:E6:6B:A5:B1 server=defconf
add address=192.168.88.240 client-id=1:78:d2:94:b5:89:cf mac-address=\
    78:D2:94:B5:89:CF server=defconf
add address=192.168.88.231 client-id=1:78:d2:94:a4:12:8e mac-address=\
    78:D2:94:A4:12:8E server=defconf
add address=192.168.88.232 client-id=1:94:a6:7e:60:cc:7f mac-address=\
    94:A6:7E:60:CC:7F server=defconf
add address=192.168.88.131 client-id=1:78:45:58:ea:e9:26 mac-address=\
    78:45:58:EA:E9:26 server=defconf
add address=192.168.88.77 client-id=1:78:45:58:e8:f:39 mac-address=\
    78:45:58:E8:0F:39 server=defconf
add address=192.168.88.22 client-id=1:b4:22:0:50:a:7f mac-address=\
    B4:22:00:50:0A:7F server=defconf
/ip dhcp-server network
add address=192.168.88.0/24 comment=defconf dns-server=192.168.88.1 gateway=\
    192.168.88.1
/ip dns
set allow-remote-requests=yes
/ip dns static
add address=192.168.88.1 comment=defconf name=router.lan
add address=192.168.88.1 name=microtik.maple.cml.me
add address=192.168.88.250 name=synology.maple.cml.me
add address=192.168.1.254 name=ridgewave.maple.cml.me
add address=192.168.88.77 name=unifi-yinlounge.maple.cml.me
add address=192.168.88.131 name=unifi-mainhall.maple.cml.me
add address=192.168.88.239 name=brother-letterhead.maple.cml.me
add address=192.168.88.22 name=brother-yinlounge.maple.cml.me
add address=192.168.88.247 name=hp-yinlounge.maple.cml.me
add address=192.168.88.250 name=status.maple.cml.me
/ip firewall address-list
add address=192.168.88.5 list=vtel_users
/ip firewall filter
add action=accept chain=input comment=\
    "defconf: accept established,related,untracked" connection-state=\
    established,related,untracked
add action=drop chain=input comment="defconf: drop invalid" connection-state=\
    invalid
add action=accept chain=input comment="defconf: accept ICMP" protocol=icmp
add action=accept chain=input comment=\
    "defconf: accept to local loopback (for CAPsMAN)" dst-address=127.0.0.1
add action=drop chain=input comment="defconf: drop all not coming from LAN" \
    in-interface-list=!LAN
add action=accept chain=forward comment="defconf: accept in ipsec policy" \
    ipsec-policy=in,ipsec
add action=accept chain=forward comment="defconf: accept out ipsec policy" \
    ipsec-policy=out,ipsec
add action=fasttrack-connection chain=forward comment="defconf: fasttrack" \
    connection-state=established,related
add action=accept chain=forward comment=\
    "defconf: accept established,related, untracked" connection-state=\
    established,related,untracked
add action=drop chain=forward comment="defconf: drop invalid" \
    connection-state=invalid
add action=drop chain=forward comment=\
    "defconf: drop all from WAN not DSTNATed" connection-nat-state=!dstnat \
    connection-state=new in-interface-list=WAN
add action=drop chain=forward comment=\
    "koshin 5-3-2022: drop all from VTEL not DSTNATed" connection-nat-state=\
    !dstnat connection-state=new in-interface-list=VTEL
/ip firewall mangle
add action=log chain=prerouting comment=\
    "koshin 5-11-2022: log packets on the special address list" log-prefix=\
    "received packet in prerouting" src-address-list=vtel_users
add action=mark-connection chain=prerouting comment=\
    "koshin 5-3-2022: mark connections bound for vtel" connection-state=new \
    new-connection-mark=route_to_vtel src-address-list=vtel_users
/ip firewall nat
add action=masquerade chain=srcnat comment="defconf: masquerade" \
    ipsec-policy=out,none out-interface-list=WAN
add action=masquerade chain=srcnat comment=\
    "koshin 5-3-2022: macsquerade for VTEL" ipsec-policy=out,none \
    out-interface-list=VTEL
/ip route
add distance=1 gateway=10.102.4.142 routing-mark=route_to_vtel
/routing filter
add chain=dynamic-in comment="koshin 5-3-2022" distance=1 set-check-gateway=\
    ping
/system clock
set time-zone-name=America/New_York
/system logging
add action=packetlog topics=firewall
/tool mac-server
set allowed-interface-list=LAN
/tool mac-server mac-winbox
set allowed-interface-list=LAN
