CC = gcc
CFLAGS = -O2
LIBS = -lcurl

all: download

download: download.c
	$(CC) $(CFLAGS) -o download download.c $(LIBS)

clean:
	rm -f download
