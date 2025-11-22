use anyhow::{Context, Result};
use inquire::{validator::Validation, Text};
use std::error::Error;

/// Prompt for a URL with validation
pub fn prompt_url() -> Result<String> {
    let url_validator = |input: &str| -> Result<Validation, Box<dyn Error + Send + Sync>> {
        if input.trim().is_empty() {
            return Ok(Validation::Invalid("URL cannot be empty".into()));
        }

        // Basic URL validation
        if url::Url::parse(input).is_ok() {
            Ok(Validation::Valid)
        } else {
            Ok(Validation::Invalid("Please enter a valid URL".into()))
        }
    };

    Text::new("URL:")
        .with_validator(url_validator)
        .prompt()
        .context("Failed to read URL input")
}

/// Prompt for an optional title
pub fn prompt_title() -> Result<Option<String>> {
    let title = Text::new("Title (optional, press Enter to skip):")
        .prompt()
        .context("Failed to read title input")?;

    let trimmed = title.trim();
    if trimmed.is_empty() {
        Ok(None)
    } else {
        Ok(Some(trimmed.to_string()))
    }
}

/// Prompt for an optional description
pub fn prompt_description() -> Result<Option<String>> {
    let description = Text::new("Description (optional, press Enter to skip):")
        .prompt()
        .context("Failed to read description input")?;

    let trimmed = description.trim();
    if trimmed.is_empty() {
        Ok(None)
    } else {
        Ok(Some(trimmed.to_string()))
    }
}
