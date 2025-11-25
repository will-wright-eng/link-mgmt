use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use std::io::IsTerminal;

use crate::client::{Link, LinkClient, UserClient};
use crate::config::Config;
use crate::prompts::{prompt_description, prompt_title, prompt_url};

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
        /// URL to save (optional - if not provided, interactive mode will prompt)
        url: Option<String>,

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
        /// Link ID (UUID)
        id: String,
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
    /// Register a new user account
    Register {
        /// Email address
        email: String,
    },
    /// Login with an API key
    Login {
        /// API key
        #[arg(short, long)]
        api_key: String,
    },
    /// Logout (remove stored API key)
    Logout,
    /// Show current user information
    Me,
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
    fn resolve_api_url(cli_api_url: Option<String>, config: &Config) -> String {
        cli_api_url
            .or_else(|| config.api_url.clone())
            .unwrap_or_else(|| "http://localhost:8000".to_string())
    }

    fn create_client(api_url: &str, config: &Config) -> Result<LinkClient> {
        LinkClient::new(api_url, config).context("Failed to create API client")
    }

    fn display_link_saved(link: &Link) {
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

    fn display_link(link: &Link, show_updated: bool) {
        println!("Link #{}:", link.id);
        println!("  URL: {}", link.url);
        if let Some(title) = &link.title {
            println!("  Title: {}", title);
        }
        if let Some(description) = &link.description {
            println!("  Description: {}", description);
        }
        println!("  Created: {}", link.created_at);
        if show_updated {
            println!("  Updated: {}", link.updated_at);
        }
    }

    fn display_links(links: &Vec<Link>, limit: Option<usize>) {
        println!("Found {} link(s):\n", links.len());
        for link in links.iter().take(limit.unwrap_or(20)) {
            println!("  [{}] {}", link.id, link.url);
            if let Some(title) = &link.title {
                println!("      Title: {}", title);
            }
            if let Some(description) = &link.description {
                let max_len = 80;
                if description.len() > max_len {
                    println!("      Description: {}...", &description[..max_len]);
                } else {
                    println!("      Description: {}", description);
                }
            }
            println!("      Created: {}", link.created_at);
            println!("      Updated: {}\n", link.updated_at);
        }
    }

    pub async fn run(self) -> Result<()> {
        let config = Config::load()?;
        let api_url = Self::resolve_api_url(self.api_url, &config);

        match self.command {
            Commands::Save {
                url,
                title,
                description,
            } => Self::handle_save(api_url, config, url, title, description).await,
            Commands::List { limit } => Self::handle_list(api_url, config, limit).await,
            Commands::Get { id } => Self::handle_get(api_url, config, id).await,
            Commands::Auth(cmd) => Self::handle_auth(api_url, config, cmd).await,
            Commands::Config(cmd) => Self::handle_config(config, cmd),
        }
    }

    async fn handle_save(
        api_url: String,
        config: Config,
        url: Option<String>,
        title: Option<String>,
        description: Option<String>,
    ) -> Result<()> {
        // Check if we're in a non-interactive environment
        let is_interactive = std::io::stdin().is_terminal();

        // Collect values, prompting for missing ones
        let final_url = match url {
            Some(u) => u,
            None => {
                if !is_interactive {
                    anyhow::bail!("URL is required when running in non-interactive mode");
                }
                prompt_url().context("Failed to get URL")?
            }
        };

        let final_title = match title {
            Some(t) => Some(t),
            None => {
                if is_interactive {
                    prompt_title().context("Failed to get title")?
                } else {
                    None
                }
            }
        };

        let final_description = match description {
            Some(d) => Some(d),
            None => {
                if is_interactive {
                    prompt_description().context("Failed to get description")?
                } else {
                    None
                }
            }
        };

        let client = Self::create_client(&api_url, &config)?;
        let link = client
            .create_link(
                &final_url,
                final_title.as_deref(),
                final_description.as_deref(),
            )
            .await
            .context("Failed to create link")?;
        Self::display_link_saved(&link);
        Ok(())
    }

    async fn handle_list(api_url: String, config: Config, limit: Option<usize>) -> Result<()> {
        let client = Self::create_client(&api_url, &config)?;
        let links = client.list_links().await.context("Failed to list links")?;

        Self::display_links(&links, limit);
        // let display_links: Vec<_> = links.iter().take(limit.unwrap_or(20)).collect();
        // println!("Found {} link(s):\n", links.len());
        // for link in display_links {
        //     println!("  [{}] {}", link.id, link.url);
        //     if let Some(title) = &link.title {
        //         println!("      Title: {}", title);
        //     }
        //     println!("      Created: {}\n", link.created_at);
        // }
        Ok(())
    }

    async fn handle_get(api_url: String, config: Config, id: String) -> Result<()> {
        let client = Self::create_client(&api_url, &config)?;
        let link = client.get_link(&id).await.context("Failed to get link")?;
        Self::display_link(&link, true);
        Ok(())
    }

    async fn handle_auth(api_url: String, config: Config, cmd: AuthCommands) -> Result<()> {
        match cmd {
            AuthCommands::Register { email } => Self::handle_register(api_url, config, email).await,
            AuthCommands::Login { api_key } => {
                config.set_api_key(&api_key)?;

                // Try to fetch and save username
                if let Ok(client) = UserClient::new(&api_url, &config) {
                    if let Ok(user) = client.get_me().await {
                        config.set_username(&user.email)?;
                        println!("✓ API key saved successfully");
                        println!("  Username: {}", user.email);
                    } else {
                        println!("✓ API key saved successfully");
                        println!("  Note: Could not fetch user info. Run 'lnk auth me' to verify.");
                    }
                } else {
                    println!("✓ API key saved successfully");
                }

                Ok(())
            }
            AuthCommands::Logout => {
                config.remove_api_key()?;
                config.remove_username()?;
                println!("✓ API key and username removed");
                Ok(())
            }
            AuthCommands::Me => Self::handle_me(api_url, config).await,
            AuthCommands::Status => Self::handle_auth_status(config),
        }
    }

    async fn handle_register(api_url: String, config: Config, email: String) -> Result<()> {
        let client = UserClient::new(&api_url, &config)?;
        let user = client
            .create_user(&email)
            .await
            .context("Failed to register user")?;

        // Automatically save the API key and username
        config.set_api_key(&user.api_key)?;
        config.set_username(&user.email)?;

        println!("✓ User registered successfully!");
        println!("  Email: {}", user.email);
        println!("  User ID: {}", user.id);
        println!("  Created: {}", user.created_at);
        println!("  API key saved automatically");
        println!("\n⚠️  Save this API key securely:");
        println!("  {}", user.api_key);

        Ok(())
    }

    async fn handle_me(api_url: String, config: Config) -> Result<()> {
        let client = UserClient::new(&api_url, &config)?;
        let user = client.get_me().await.context("Failed to get user info")?;

        // Update username in config if it's different
        config.set_username(&user.email)?;

        println!("Current user:");
        println!("  ID: {}", user.id);
        println!("  Email: {}", user.email);
        println!("  Created: {}", user.created_at);
        println!("  Updated: {}", user.updated_at);

        Ok(())
    }

    fn handle_auth_status(config: Config) -> Result<()> {
        match config.get_api_key()? {
            Some(key) => {
                println!("✓ Authenticated");
                if let Some(username) = config.get_username()? {
                    println!("  Username: {}", username);
                }
                println!(
                    "  API key: {}...{}",
                    &key[..8.min(key.len())],
                    &key[key.len().saturating_sub(4)..]
                );
                Ok(())
            }
            None => {
                println!("✗ Not authenticated");
                println!("  Run 'lnk auth login --api-key <key>' to authenticate");
                Ok(())
            }
        }
    }

    fn handle_config(config: Config, cmd: ConfigCommands) -> Result<()> {
        match cmd {
            ConfigCommands::Set { key, value } => {
                config.set(&key, &value)?;
                println!("✓ Configuration updated: {} = {}", key, value);
                Ok(())
            }
            ConfigCommands::Get { key } => {
                match config.get(&key)? {
                    Some(val) => println!("{}", val),
                    None => println!("Configuration key '{}' not found", key),
                }
                Ok(())
            }
        }
    }
}
