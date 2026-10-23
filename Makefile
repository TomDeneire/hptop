build:
	go build -o hptop .
	mv hptop $$HOME/bin
run: build
	./hptop
