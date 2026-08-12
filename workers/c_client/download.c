#include <stdio.h>
#include <stdlib.h>
#include <curl/curl.h>

struct FData { FILE *fp; };

static size_t write_cb(void *ptr, size_t size, size_t nmemb, void *userdata) {
    struct FData *d = (struct FData*)userdata;
    return fwrite(ptr, size, nmemb, d->fp);
}

int main(int argc, char **argv) {
    if(argc != 3) {
        fprintf(stderr, "Usage: %s <url> <outpath>\n", argv[0]);
        return 2;
    }
    const char *url = argv[1];
    const char *out = argv[2];
    CURL *c = curl_easy_init();
    if(!c) { fprintf(stderr, "curl init failed\n"); return 1; }
    FILE *fp = fopen(out, "wb");
    if(!fp) { fprintf(stderr, "failed to open output file\n"); return 1; }
    struct FData d = { .fp = fp };
    curl_easy_setopt(c, CURLOPT_URL, url);
    curl_easy_setopt(c, CURLOPT_WRITEFUNCTION, write_cb);
    curl_easy_setopt(c, CURLOPT_WRITEDATA, &d);
    CURLcode res = curl_easy_perform(c);
    curl_easy_cleanup(c);
    fclose(fp);
    if(res != CURLE_OK) {
        fprintf(stderr, "curl error: %s\n", curl_easy_strerror(res));
        return 1;
    }
    printf("saved %s\n", out);
    return 0;
}
