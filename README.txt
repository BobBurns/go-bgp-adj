go-bgp-adj
==========

Simple web page to display BGP neighbor adjacency states
by querying AKiPS API. 

Set up and run
==============

$ echo api-password > apass

change line : 
  akipsURL := "https://put-your-url-here/api-db?password=" + pass

$ go build

$ ./go-bgp-adj

access page at http://localhost:8082
