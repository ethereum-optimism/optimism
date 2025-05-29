use crate::auth::AuthenticationParseError::{
    DuplicateAPIKeyArgument, DuplicateApplicationArgument, MissingAPIKeyArgument,
    MissingApplicationArgument, NoData, TooManyComponents,
};
use std::collections::{HashMap, HashSet};

#[derive(Clone, Debug)]
pub struct Authentication {
    key_to_application: HashMap<String, String>,
}

#[derive(Debug, PartialEq)]
pub enum AuthenticationParseError {
    NoData(),
    MissingApplicationArgument(String),
    MissingAPIKeyArgument(String),
    TooManyComponents(String),
    DuplicateApplicationArgument(String),
    DuplicateAPIKeyArgument(String),
}

impl std::fmt::Display for AuthenticationParseError {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            NoData() => write!(f, "No API Keys Provided"),
            MissingApplicationArgument(arg) => write!(f, "Missing application argument: [{}]", arg),
            MissingAPIKeyArgument(app) => write!(f, "Missing API Key argument: [{}]", app),
            TooManyComponents(app) => write!(f, "Too many components: [{}]", app),
            DuplicateApplicationArgument(app) => {
                write!(f, "Duplicate application argument: [{}]", app)
            }
            DuplicateAPIKeyArgument(app) => write!(f, "Duplicate API key: [{}]", app),
        }
    }
}

impl std::error::Error for AuthenticationParseError {}

impl TryFrom<Vec<String>> for Authentication {
    type Error = AuthenticationParseError;

    fn try_from(args: Vec<String>) -> Result<Self, Self::Error> {
        let mut applications = HashSet::new();
        let mut key_to_application: HashMap<String, String> = HashMap::new();

        if args.is_empty() {
            return Err(NoData());
        }

        for arg in args {
            let mut parts = arg.split(":");
            let app = parts
                .next()
                .ok_or(MissingApplicationArgument(arg.clone()))?;
            if app.is_empty() {
                return Err(MissingApplicationArgument(arg.clone()));
            }

            let key = parts.next().ok_or(MissingAPIKeyArgument(app.to_string()))?;
            if key.is_empty() {
                return Err(MissingAPIKeyArgument(app.to_string()));
            }

            if parts.count() > 0 {
                return Err(TooManyComponents(app.to_string()));
            }

            if applications.contains(app) {
                return Err(DuplicateApplicationArgument(app.to_string()));
            }

            if key_to_application.contains_key(key) {
                return Err(DuplicateAPIKeyArgument(app.to_string()));
            }

            applications.insert(app.to_string());
            key_to_application.insert(key.to_string(), app.to_string());
        }

        Ok(Self { key_to_application })
    }
}

impl Authentication {
    pub fn none() -> Self {
        Self {
            key_to_application: HashMap::new(),
        }
    }

    #[allow(dead_code)]
    pub fn new(api_keys: HashMap<String, String>) -> Self {
        Self {
            key_to_application: api_keys,
        }
    }

    pub fn get_application_for_key(&self, api_key: &String) -> Option<&String> {
        self.key_to_application.get(api_key)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parsing() {
        let auth = Authentication::try_from(vec![
            "app1:key1".to_string(),
            "app2:key2".to_string(),
            "app3:key3".to_string(),
        ])
        .unwrap();

        assert_eq!(auth.key_to_application.len(), 3);
        assert_eq!(auth.key_to_application["key1"], "app1");
        assert_eq!(auth.key_to_application["key2"], "app2");
        assert_eq!(auth.key_to_application["key3"], "app3");

        let auth = Authentication::try_from(vec![
            "app1:key1".to_string(),
            "".to_string(),
            "app3:key3".to_string(),
        ]);
        assert!(auth.is_err());
        assert_eq!(auth.unwrap_err(), MissingApplicationArgument("".into()));

        let auth = Authentication::try_from(vec![
            "app1:key1".to_string(),
            "app2".to_string(),
            "app3:key3".to_string(),
        ]);
        assert!(auth.is_err());
        assert_eq!(auth.unwrap_err(), MissingAPIKeyArgument("app2".into()));

        let auth = Authentication::try_from(vec![
            "app1:key1".to_string(),
            ":".to_string(),
            "app3:key3".to_string(),
        ]);
        assert!(auth.is_err());
        assert_eq!(auth.unwrap_err(), MissingApplicationArgument(":".into()));

        let auth = Authentication::try_from(vec![
            "app1:key1".to_string(),
            "app2:".to_string(),
            "app3:key3".to_string(),
        ]);
        assert!(auth.is_err());
        assert_eq!(auth.unwrap_err(), MissingAPIKeyArgument("app2".into()));

        let auth = Authentication::try_from(vec![
            "app1:key1".to_string(),
            "app2:key2:unexpected2".to_string(),
            "app3:key3".to_string(),
        ]);
        assert!(auth.is_err());
        assert_eq!(auth.unwrap_err(), TooManyComponents("app2".into()));

        let auth = Authentication::try_from(vec![
            "app1:key1".to_string(),
            "app1:key3".to_string(),
            "app2:key2".to_string(),
        ]);
        assert!(auth.is_err());
        assert_eq!(
            auth.unwrap_err(),
            DuplicateApplicationArgument("app1".into())
        );

        let auth = Authentication::try_from(vec!["app1:key1".to_string(), "app2:key1".to_string()]);
        assert!(auth.is_err());
        assert_eq!(auth.unwrap_err(), DuplicateAPIKeyArgument("app2".into()));
    }
}
