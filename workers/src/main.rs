use reqwest::Client;
use std::env;
use std::fs::File;
use std::io::Write;

// Simple Rust downloader CLI to save files locally from a URL.
// Usage: `cargo run -- download <url> <outpath>`

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut args = env::args().skip(1);
    match args.next().as_deref() {
        Some("download") => {
            let url = args.next().expect("missing url");
            let out = args.next().expect("missing output path");
            download_file(&url, &out).await?;
            println!("saved {}", out);
        }
        _ => {
            println!("fileforce-worker: available commands:\n  download <url> <outpath>");
        }
    }
    Ok(())
}

async fn download_file(url: &str, out_path: &str) -> Result<(), Box<dyn std::error::Error>> {
    let client = Client::new();
    let resp = client.get(url).send().await?;
    if !resp.status().is_success() {
        return Err(format!("failed to download: {}", resp.status()).into());
    }
    let bytes = resp.bytes().await?;
    let mut file = File::create(out_path)?;
    file.write_all(&bytes)?;
    Ok(())
}
