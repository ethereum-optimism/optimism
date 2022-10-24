
import os
import urllib.request

dashboard_list=[
  {
   'name': 'Geth Optimism',
   'filename': 'single_geth.json',
   'url': 'https://grafana.com/api/dashboards/13877/revisions/1/download',
   'datasource': 'InfluxDB'
  },
  {
   'name': 'Geth Ethereum',
   'filename': 'single_geth_eth.json',
   'url': 'https://grafana.com/api/dashboards/13877/revisions/1/download',
   'datasource': 'InfluxDB_eth'
  }
]
dashboard_path="/grafana-dashboards"

GF_SECURITY_ADMIN_PASSWORD = os.environ.get('GF_SECURITY_ADMIN_PASSWORD')
if GF_SECURITY_ADMIN_PASSWORD is None:
  print('GF_SECURITY_ADMIN_PASSWORD env value is missing, exiting.')
  sys.exit(1)

if (not os.path.exists(dashboard_path)) or (not os.path.isdir(dashboard_path)) or (not os.access(dashboard_path, os.W_OK)):
  print('Dashboard path %s is not writable, exiting'.format(dashboard_path))
  sys.exit(1)

for dashboard in dashboard_list:
  with urllib.request.urlopen(dashboard['url']) as f:
    response = f.read()
    decoded_html = response.decode('utf-8')
    data = decoded_html.replace('${DS_INFLUXDB}', dashboard['datasource'])
    data = data.replace("Geth Dashboard", dashboard['name'])
    data = data.replace("QC1Arp5Wk", "QC1Arp5Wk"+dashboard['datasource'])
    d_file = open(os.path.join(dashboard_path, dashboard['filename']),'w')
    d_file.write(data)
    d_file.close()
