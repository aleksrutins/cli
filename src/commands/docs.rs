use anyhow::bail;
use is_terminal::IsTerminal;

use crate::{
    consts::{ABORTED_BY_USER, NON_INTERACTIVE_FAILURE},
    interact_or,
    util::prompt::prompt_confirm_with_default,
};

use super::*;

/// Open Railway Documentation in default browser
#[derive(Parser)]
pub struct Args {}

pub async fn command(_args: Args, _json: bool) -> Result<()> {
    interact_or!(NON_INTERACTIVE_FAILURE);

    let confirm = prompt_confirm_with_default("Open the browser?", true)?;

    if !confirm {
        bail!(ABORTED_BY_USER);
    }

    ::open::that("https://docs.railway.app/")?;
    Ok(())
}
