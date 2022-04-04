Build with :

CGO_ENABLED=0 go build -o ./build/kind -a -ldflags '-extldflags "-static"' .

and then run 


kind create cluster 
