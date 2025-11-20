use anyhow::{Context, Result};
use clap::{Parser, Subcommand};

use crate::client::LinkClient;
use crate::config::Config;

#[derive(Parser)]
#[command(name = "lnk")]
#[command(about = "A CLI client for link management", long_about = None)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,

    /// API base URL (overrides config)
    #[arg(long, env = "LNK_API_URL")]
    pub api_url: Option<String>,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Save a link to the API
    Save {
        /// URL to save
        url: String,

        /// Optional title for the link
        #[arg(short, long)]
        title: Option<String>,

        /// Optional description for the link
        #[arg(short, long)]
        description: Option<String>,
    },

    /// List all links
    List {
        /// Limit the number of results
        #[arg(short, long, default_value = "20")]
        limit: Option<usize>,
    },

    /// Get a specific link by ID
    Get {
        /// Link ID
        id: u64,
    },

    /// Authentication commands
    #[command(subcommand)]
    Auth(AuthCommands),

    /// Configuration commands
    #[command(subcommand)]
    Config(ConfigCommands),
}

#[derive(Subcommand)]
pub enum AuthCommands {
    /// Login with an API key
    Login {
        /// API key
        #[arg(short, long)]
        api_key: String,
    },
    /// Logout (remove stored API key)
    Logout,
    /// Show authentication status
    Status,
}

#[derive(Subcommand)]
pub enum ConfigCommands {
    /// Set a configuration value
    Set {
        /// Configuration key (e.g., api_url)
        key: String,
        /// Configuration value
        value: String,
    },
    /// Get a configuration value
    Get {
        /// Configuration key
        key: String,
    },
}

impl Cli {
    pub async fn run(self) -> Result<()> {
        let config = Config::load()?;
        let api_url = self
            .api_url
            .or_else(|| config.api_url.clone())
            .unwrap_or_else(|| "http://localhost:8000".to_string());

        match self.command {
            Commands::Save {
                url,
                title,
                description,
            } => {
                let client = LinkClient::new(&api_url, &config)?;
                let link = client
                    .create_link(&url, title.as_deref(), description.as_deref())
                    .await
                    .context("Failed to create link")?;
                println!("✓ Link saved successfully!");
                println!("  ID: {}", link.id);
                println!("  URL: {}", link.url);
                if let Some(title) = &link.title {
                    println!("  Title: {}", title);
                }
                if let Some(description) = &link.description {
                    println!("  Description: {}", description);
                }
                println!("  Created: {}", link.created_at);
            }
            Commands::List { limit } => {
                let client = LinkClient::new(&api_url, &config)?;
                let links = client.list_links().await.context("Failed to list links")?;

                let display_links: Vec<_> = links.iter().take(limit.unwrap_or(20)).collect();
                println!("Found {} link(s):\n", links.len());
                for link in display_links {
                    println!("  [{}] {}", link.id, link.url);
                    if let Some(title) = &link.title {
                        println!("      Title: {}", title);
                    }
                    println!("      Created: {}\n", link.created_at);
                }
            }
            Commands::Get { id } => {
                let client = LinkClient::new(&api_url, &config)?;
                let link = client.get_link(id).await.context("Failed to get link")?;
                println!("Link #{}:", link.id);
                println!("  URL: {}", link.url);
                if let Some(title) = &link.title {
                    println!("  Title: {}", title);
                }
                if let Some(description) = &link.description {
                    println!("  Description: {}", description);
                }
                println!("  Created: {}", link.created_at);
                println!("  Updated: {}", link.updated_at);
            }
            Commands::Auth(cmd) => match cmd {
                AuthCommands::Login { api_key } => {
                    config.set_api_key(&api_key)?;
                    println!("✓ API key saved successfully");
                }
                AuthCommands::Logout => {
                    config.remove_api_key()?;
                    println!("✓ API key removed");
                }
                AuthCommands::Status => match config.get_api_key()? {
                    Some(key) => {
                        println!("✓ Authenticated");
                        println!(
                            "  API key: {}...{}",
                            &key[..8.min(key.len())],
                            &key[key.len().saturating_sub(4)..]
                        );
                    }
                    None => {
                        println!("✗ Not authenticated");
                        println!("  Run 'lnk auth login --api-key <key>' to authenticate");
                    }
                },
            },
            Commands::Config(cmd) => match cmd {
                ConfigCommands::Set { key, value } => {
                    config.set(&key, &value)?;
                    println!("✓ Configuration updated: {} = {}", key, value);
                }
                ConfigCommands::Get { key } => match config.get(&key)? {
                    Some(val) => println!("{}", val),
                    None => println!("Configuration key '{}' not found", key),
                },
            },
        }

        Ok(())
    }
}
