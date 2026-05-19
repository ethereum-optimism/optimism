use proc_macro::TokenStream;
use quote::{ToTokens, quote};
use syn::{Expr, ItemFn, Meta, Token, parse_macro_input, punctuated::Punctuated};

// Define all variant information in one place
struct VariantInfo {
    name: &'static str,
    builder_type: &'static str,
    default_instance_call: &'static str,
    args_modifier: fn(&proc_macro2::TokenStream) -> proc_macro2::TokenStream,
    default_args_factory: fn() -> proc_macro2::TokenStream,
}

const BUILDER_VARIANTS: &[VariantInfo] = &[
    VariantInfo {
        name: "standard",
        builder_type: "crate::builders::StandardBuilder",
        default_instance_call: "crate::tests::LocalInstance::standard().await?",
        args_modifier: |args| quote! { #args },
        default_args_factory: || quote! { Default::default() },
    },
    VariantInfo {
        name: "flashblocks",
        builder_type: "crate::builders::FlashblocksBuilder",
        default_instance_call: "crate::tests::LocalInstance::flashblocks().await?",
        args_modifier: |args| {
            quote! {
                {
                    let mut args = #args;
                    args.flashblocks.enabled = true;
                    args.flashblocks.flashblocks_port = crate::tests::get_available_port();
                    args
                }
            }
        },
        default_args_factory: || {
            quote! {
                {
                    let mut args = crate::args::OpRbuilderArgs::default();
                    args.flashblocks.enabled = true;
                    args.flashblocks.flashblocks_port = crate::tests::get_available_port();
                    args
                }
            }
        },
    },
];

fn get_variant_info(variant: &str) -> Option<&'static VariantInfo> {
    BUILDER_VARIANTS.iter().find(|v| v.name == variant)
}

fn get_variant_names() -> Vec<&'static str> {
    BUILDER_VARIANTS.iter().map(|v| v.name).collect()
}

struct TestConfig {
    variants: std::collections::HashMap<String, Option<Expr>>, // variant name -> custom expression (None = default)
    args: Option<Expr>,   // Expression to pass to LocalInstance::new()
    config: Option<Expr>, // NodeConfig<OpChainSpec> for new_with_config
    multi_threaded: bool, // Whether to use multi_thread flavor
}

impl syn::parse::Parse for TestConfig {
    fn parse(input: syn::parse::ParseStream) -> syn::Result<Self> {
        let mut config = TestConfig {
            variants: std::collections::HashMap::new(),
            args: None,
            config: None,
            multi_threaded: false,
        };

        if input.is_empty() {
            // No arguments provided, generate all variants with defaults
            for variant in BUILDER_VARIANTS {
                config.variants.insert(variant.name.to_string(), None);
            }
            return Ok(config);
        }

        let args: Punctuated<Meta, Token![,]> = input.parse_terminated(Meta::parse, Token![,])?;
        let variant_names = get_variant_names();

        for arg in args {
            match arg {
                Meta::Path(path) => {
                    if let Some(ident) = path.get_ident() {
                        let name = ident.to_string();
                        if variant_names.contains(&name.as_str()) {
                            config.variants.insert(name, None);
                        } else if name == "multi_threaded" {
                            config.multi_threaded = true;
                        } else {
                            return Err(syn::Error::new_spanned(
                                path,
                                format!(
                                    "Unknown variant '{}'. Use one of: {}, 'multi_threaded', 'args', or 'config'",
                                    name,
                                    variant_names.join(", ")
                                ),
                            ));
                        }
                    }
                }
                Meta::NameValue(nv) => {
                    if let Some(ident) = nv.path.get_ident() {
                        let name = ident.to_string();
                        if variant_names.contains(&name.as_str()) {
                            config.variants.insert(name, Some(nv.value));
                        } else if name == "args" {
                            config.args = Some(nv.value);
                        } else if name == "config" {
                            config.config = Some(nv.value);
                        } else {
                            return Err(syn::Error::new_spanned(
                                nv.path,
                                format!(
                                    "Unknown attribute '{}'. Use one of: {}, 'multi_threaded', 'args', or 'config'",
                                    name,
                                    variant_names.join(", ")
                                ),
                            ));
                        }
                    }
                }
                _ => {
                    return Err(syn::Error::new_spanned(
                        arg,
                        format!(
                            "Invalid attribute format. Use one of: {}, 'multi_threaded', 'args', or 'config'",
                            variant_names.join(", ")
                        ),
                    ));
                }
            }
        }

        // Validate that custom expressions and args/config are not used together
        for (variant, custom_expr) in &config.variants {
            if custom_expr.is_some() && (config.args.is_some() || config.config.is_some()) {
                return Err(syn::Error::new_spanned(
                    config.args.as_ref().or(config.config.as_ref()).unwrap(),
                    format!(
                        "Cannot use 'args' or 'config' with custom '{variant}' expression. Use either '{variant} = expression' or 'args/config' parameters, not both."
                    ),
                ));
            }
        }

        // If only args/config/multi_threaded is specified, generate all variants
        if config.variants.is_empty()
            && (config.args.is_some() || config.config.is_some() || config.multi_threaded)
        {
            for variant in BUILDER_VARIANTS {
                config.variants.insert(variant.name.to_string(), None);
            }
        }

        Ok(config)
    }
}

fn generate_instance_init(
    variant: &str,
    custom_expr: Option<&Expr>,
    args: &Option<Expr>,
    config: &Option<Expr>,
) -> proc_macro2::TokenStream {
    if let Some(expr) = custom_expr {
        return quote! { #expr };
    }

    let variant_info =
        get_variant_info(variant).unwrap_or_else(|| panic!("Unknown variant: {variant}"));

    let builder_type: proc_macro2::TokenStream = variant_info.builder_type.parse().unwrap();
    let default_call: proc_macro2::TokenStream =
        variant_info.default_instance_call.parse().unwrap();

    match (args, config) {
        (None, None) => default_call,
        (Some(args_expr), None) => {
            let modified_args = (variant_info.args_modifier)(&quote! { #args_expr });
            quote! { crate::tests::LocalInstance::new::<#builder_type>(#modified_args).await? }
        }
        (None, Some(config_expr)) => {
            let default_args = (variant_info.default_args_factory)();
            quote! {
                crate::tests::LocalInstance::new_with_config::<#builder_type>(#default_args, #config_expr).await?
            }
        }
        (Some(args_expr), Some(config_expr)) => {
            let modified_args = (variant_info.args_modifier)(&quote! { #args_expr });
            quote! {
                crate::tests::LocalInstance::new_with_config::<#builder_type>(#modified_args, #config_expr).await?
            }
        }
    }
}

#[proc_macro_attribute]
pub fn rb_test(args: TokenStream, input: TokenStream) -> TokenStream {
    let input_fn = parse_macro_input!(input as ItemFn);
    let config = parse_macro_input!(args as TestConfig);

    validate_signature(&input_fn);

    // Create the original function without test attributes (helper function)
    let mut helper_fn = input_fn.clone();
    helper_fn
        .attrs
        .retain(|attr| !attr.path().is_ident("test") && !attr.path().is_ident("tokio"));

    let original_name = &input_fn.sig.ident;
    let mut generated_functions = vec![quote! { #helper_fn }];

    // Generate test for each requested variant
    for (variant, custom_expr) in &config.variants {
        let test_name =
            syn::Ident::new(&format!("{original_name}_{variant}"), original_name.span());

        let instance_init =
            generate_instance_init(variant, custom_expr.as_ref(), &config.args, &config.config);

        let test_attribute = if config.multi_threaded {
            quote! { #[tokio::test(flavor = "multi_thread")] }
        } else {
            quote! { #[tokio::test] }
        };

        generated_functions.push(quote! {
            #test_attribute
            async fn #test_name() -> eyre::Result<()> {
                let subscriber = tracing_subscriber::fmt()
                    .with_env_filter(std::env::var("RUST_LOG")
                        .unwrap_or_else(|_| "info".to_string()))
                    .with_file(true)
                    .with_line_number(true)
                    .with_test_writer()
                    .finish();
                let _guard = tracing::subscriber::set_global_default(subscriber);
                tracing::info!("{} start", stringify!(#test_name));

                let instance = #instance_init;
                #original_name(instance).await
            }
        });
    }

    TokenStream::from(quote! {
        #(#generated_functions)*
    })
}

fn validate_signature(item_fn: &ItemFn) {
    if item_fn.sig.asyncness.is_none() {
        panic!("Function must be async.");
    }
    if item_fn.sig.inputs.len() != 1 {
        panic!("Function must have exactly one parameter of type LocalInstance.");
    }

    let output_types = item_fn
        .sig
        .output
        .to_token_stream()
        .to_string()
        .replace(" ", "");

    if output_types != "->eyre::Result<()>" {
        panic!("Function must return Result<(), eyre::Error>. Actual: {output_types}",);
    }
}

// Generate conditional execution macros for each variant
macro_rules! generate_if_variant_macros {
    ($($variant_name:ident),*) => {
        $(
            paste::paste! {
                #[proc_macro]
                pub fn [<if_ $variant_name>](input: TokenStream) -> TokenStream {
                    let input = proc_macro2::TokenStream::from(input);
                    let suffix = concat!("_", stringify!($variant_name));

                    TokenStream::from(quote! {
                        if std::thread::current().name().unwrap_or("").ends_with(#suffix) {
                            #input
                        }
                    })
                }
            }
        )*
    };
}

// Generate macros for all variants
generate_if_variant_macros!(standard, flashblocks);
