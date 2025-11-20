use anyhow::Result;
use clap::Parser;

mod cli;
mod client;
mod config;
// TODO: Recreate display module for table formatting
// mod display;
// TODO: Recreate utils module for URL validation and utilities
// mod utils;

use cli::Cli;

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    cli.run().await
}
