use anyhow::{Context, Result};
use keyring::Entry;
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

const SERVICE_NAME: &str = "lnk-cli";
const CONFIG_FILE: &str = "config.toml";

#[derive(Debug, Clone)]
pub struct Config {
    config_dir: PathBuf,
    pub api_url: Option<String>,
}

impl Config {
    pub fn load() -> Result<Self> {
        let config_dir = Self::get_config_dir()?;
        fs::create_dir_all(&config_dir).context("Failed to create config directory")?;

        let config_file = config_dir.join(CONFIG_FILE);
        let api_url = if config_file.exists() {
            let content = fs::read_to_string(&config_file).context("Failed to read config file")?;
            let config: HashMap<String, toml::Value> =
                toml::from_str(&content).context("Failed to parse config file")?;
            config
                .get("api")
                .and_then(|v| v.get("url"))
                .and_then(|v| v.as_str())
                .map(|s| s.to_string())
        } else {
            None
        };

        Ok(Self {
            config_dir,
            api_url,
        })
    }

    pub fn get_config_dir() -> Result<PathBuf> {
        dirs::config_dir()
            .map(|d| d.join("lnk"))
            .context("Failed to determine config directory")
    }

    pub fn get_api_key(&self) -> Result<Option<String>> {
        let entry = Entry::new(SERVICE_NAME, "api_key")?;
        match entry.get_password() {
            Ok(key) => Ok(Some(key)),
            Err(keyring::Error::NoEntry) => Ok(None),
            Err(e) => Err(anyhow::anyhow!("Failed to get API key: {}", e)),
        }
    }

    pub fn set_api_key(&self, api_key: &str) -> Result<()> {
        let entry = Entry::new(SERVICE_NAME, "api_key")?;
        entry
            .set_password(api_key)
            .context("Failed to store API key")?;
        Ok(())
    }

    pub fn remove_api_key(&self) -> Result<()> {
        let entry = Entry::new(SERVICE_NAME, "api_key")?;
        match entry.delete_password() {
            Ok(()) => Ok(()),
            Err(keyring::Error::NoEntry) => Ok(()), // Already removed
            Err(e) => Err(anyhow::anyhow!("Failed to remove API key: {}", e)),
        }
    }

    pub fn set(&self, key: &str, value: &str) -> Result<()> {
        let config_file = self.config_dir.join(CONFIG_FILE);
        let mut config: HashMap<String, toml::Value> = if config_file.exists() {
            let content = fs::read_to_string(&config_file).context("Failed to read config file")?;
            toml::from_str(&content).unwrap_or_default()
        } else {
            HashMap::new()
        };

        // Handle nested keys like "api.url"
        if key.contains('.') {
            let parts: Vec<&str> = key.splitn(2, '.').collect();
            if parts.len() == 2 {
                let section = parts[0];
                let subkey = parts[1];

                let section_map = config
                    .entry(section.to_string())
                    .or_insert_with(|| toml::Value::Table(toml::value::Table::new()))
                    .as_table_mut()
                    .context("Invalid config structure")?;

                section_map.insert(subkey.to_string(), toml::Value::String(value.to_string()));
            }
        } else {
            config.insert(key.to_string(), toml::Value::String(value.to_string()));
        }

        let content = toml::to_string_pretty(&config).context("Failed to serialize config")?;
        fs::write(&config_file, content).context("Failed to write config file")?;

        Ok(())
    }

    pub fn get(&self, key: &str) -> Result<Option<String>> {
        let config_file = self.config_dir.join(CONFIG_FILE);
        if !config_file.exists() {
            return Ok(None);
        }

        let content = fs::read_to_string(&config_file).context("Failed to read config file")?;
        let config: HashMap<String, toml::Value> =
            toml::from_str(&content).context("Failed to parse config file")?;

        // Handle nested keys like "api.url"
        if key.contains('.') {
            let parts: Vec<&str> = key.splitn(2, '.').collect();
            if parts.len() == 2 {
                let section = parts[0];
                let subkey = parts[1];

                if let Some(section_val) = config.get(section) {
                    if let Some(table) = section_val.as_table() {
                        if let Some(value) = table.get(subkey) {
                            return Ok(value.as_str().map(|s| s.to_string()));
                        }
                    }
                }
            }
        } else if let Some(value) = config.get(key) {
            return Ok(value.as_str().map(|s| s.to_string()));
        }

        Ok(None)
    }
}
