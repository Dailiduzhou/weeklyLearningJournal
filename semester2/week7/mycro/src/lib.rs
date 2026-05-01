use proc_macro::TokenStream;
use quote::quote;
use syn::{parse_macro_input, DeriveInput};

/// Derive macro that generates a `describe()` method
/// returning the struct name and field names.
#[proc_macro_derive(Describe)]
pub fn derive_describe(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as DeriveInput);
    let name = &input.ident;
    let name_str = name.to_string();

    // Extract field names (only for structs with named fields)
    let fields = match &input.data {
        syn::Data::Struct(data) => data
            .fields
            .iter()
            .filter_map(|f| f.ident.as_ref())
            .map(|id| id.to_string())
            .collect::<Vec<_>>(),
        _ => vec![],
    };

    let field_list = fields.join(", ");

    let expanded = quote! {
        impl #name {
            pub fn describe() -> String {
                format!("{} {{ {} }}", #name_str, #field_list)
            }
        }
    };

    TokenStream::from(expanded)
}
