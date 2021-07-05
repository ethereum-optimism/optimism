import sys
import os

try:
    # Python 3
    from http.server import HTTPServer, SimpleHTTPRequestHandler
except ImportError: 
    # Python 2
    from BaseHTTPServer import HTTPServer
    from SimpleHTTPServer import SimpleHTTPRequestHandler

class CORSRequestHandler (SimpleHTTPRequestHandler):
    def end_headers (self):
        self.send_header('Access-Control-Allow-Origin', '*')
        SimpleHTTPRequestHandler.end_headers(self)

port_a = int(sys.argv[1])
host_a = sys.argv[2]
dump_dir = sys.argv[3]

#change to directory we want to serve
os.chdir(dump_dir)

if __name__ == '__main__':
    print('Serving HTTP at host %s, port %s, and directory %s' % (host_a, port_a, dump_dir))
    httpd = HTTPServer((host_a, port_a), CORSRequestHandler)
    httpd.serve_forever()