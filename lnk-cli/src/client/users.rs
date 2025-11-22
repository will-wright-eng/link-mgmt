use anyhow::{Context, Result};
use chrono::{DateTime, TimeZone, Utc};
use reqwest::Client;
use serde::{Deserialize, Deserializer, Serialize};

use crate::config::Config;

fn deserialize_datetime<'de, D>(deserializer: D) -> Result<DateTime<Utc>, D::Error>
where
    D: Deserializer<'de>,
{
    let s = String::deserialize(deserializer)?;
    // FastAPI returns ISO 8601 format like "2025-11-20T05:28:45.444128"
    // Try parsing as RFC3339 first (handles timezone-aware strings)
    if let Ok(dt) = DateTime::parse_from_rfc3339(&s) {
        return Ok(dt.with_timezone(&Utc));
    }

    // If no timezone info, assume UTC and parse manually
    // Format: "2025-11-20T05:28:45.444128" or "2025-11-20T05:28:45"
    let formats = [
        "%Y-%m-%dT%H:%M:%S%.f", // With microseconds
        "%Y-%m-%dT%H:%M:%S",    // Without microseconds
    ];

    for format in &formats {
        if let Ok(dt) = chrono::NaiveDateTime::parse_from_str(&s, format) {
            return Ok(Utc.from_utc_datetime(&dt));
        }
    }

    Err(serde::de::Error::custom(format!(
        "Failed to parse datetime: {}",
        s
    )))
}

#[derive(Debug, Clone, Deserialize)]
pub struct User {
    pub id: String, // UUID as string
    pub email: String,
    #[serde(deserialize_with = "deserialize_datetime")]
    pub created_at: DateTime<Utc>,
    #[serde(deserialize_with = "deserialize_datetime")]
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct UserWithApiKey {
    pub id: String, // UUID as string
    pub email: String,
    pub api_key: String,
    #[serde(deserialize_with = "deserialize_datetime")]
    pub created_at: DateTime<Utc>,
    #[serde(deserialize_with = "deserialize_datetime")]
    #[allow(dead_code)] // Same as created_at for new users, kept for API response completeness
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Serialize)]
struct UserCreate {
    email: String,
}

pub struct UserClient {
    client: Client,
    base_url: String,
    api_key: Option<String>,
}

impl UserClient {
    pub fn new(base_url: &str, config: &Config) -> Result<Self> {
        let client = Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()
            .context("Failed to create HTTP client")?;

        let api_key = config.get_api_key()?;

        Ok(Self {
            client,
            base_url: base_url.trim_end_matches('/').to_string(),
            api_key,
        })
    }

    fn build_request(&self, method: reqwest::Method, path: &str) -> reqwest::RequestBuilder {
        let url = format!("{}/api/users{}", self.base_url, path);
        let mut request = self.client.request(method, &url);

        if let Some(api_key) = &self.api_key {
            request = request.header("X-API-Key", api_key);
        }

        request
    }

    pub async fn create_user(&self, email: &str) -> Result<UserWithApiKey> {
        let payload = UserCreate {
            email: email.to_string(),
        };

        let response = self
            .build_request(reqwest::Method::POST, "")
            .json(&payload)
            .send()
            .await
            .context("Failed to send request")?;

        if !response.status().is_success() {
            let status = response.status();
            let error_text = response
                .text()
                .await
                .unwrap_or_else(|_| "Unknown error".to_string());
            anyhow::bail!("API error ({}): {}", status, error_text);
        }

        let user: UserWithApiKey = response.json().await.context("Failed to parse response")?;

        Ok(user)
    }

    pub async fn get_me(&self) -> Result<User> {
        let response = self
            .build_request(reqwest::Method::GET, "/me")
            .send()
            .await
            .context("Failed to send request")?;

        if !response.status().is_success() {
            let status = response.status();
            let error_text = response
                .text()
                .await
                .unwrap_or_else(|_| "Unknown error".to_string());
            anyhow::bail!("API error ({}): {}", status, error_text);
        }

        let user: User = response.json().await.context("Failed to parse response")?;

        Ok(user)
    }
}
