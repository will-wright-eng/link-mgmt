use anyhow::Result;
use clap::Parser;

mod cli;
mod client;
mod config;
mod display;
mod utils;

use cli::Cli;

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    cli.run().await
}
