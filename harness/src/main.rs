mod cmd;
mod util;

use clap::{Args, Parser, Subcommand};
use std::path::PathBuf;
use util::{CommandError, OutputFormat};

#[derive(Parser)]
#[command(name = "harnesscli")]
#[command(about = "Harness engineering CLI for this repository")]
struct Cli {
    #[arg(long, global = true, value_enum)]
    output: Option<OutputFormat>,
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    Init,
    Boot(BootArgs),
    Smoke,
    Test,
    Lint,
    Typecheck,
    Audit { path: Option<PathBuf> },
    Cleanup(CleanupArgs),
    Observability(ObservabilityArgs),
}

#[derive(Args)]
struct BootArgs {
    #[command(subcommand)]
    command: BootCommand,
}

#[derive(Subcommand)]
enum BootCommand {
    Start,
    Status,
    Stop,
}

#[derive(Args)]
struct CleanupArgs {
    #[command(subcommand)]
    command: CleanupCommand,
}

#[derive(Subcommand)]
enum CleanupCommand {
    Scan,
    Grade,
    Fix,
}

#[derive(Args)]
struct ObservabilityArgs {
    #[command(subcommand)]
    command: ObservabilityCommand,
}

#[derive(Subcommand)]
enum ObservabilityCommand {
    Start,
    Stop,
    Query {
        #[arg(long, default_value = "logs")]
        kind: String,
        #[arg(long)]
        query: Option<String>,
    },
}

fn main() {
    let cli = Cli::parse();
    let format = util::resolve_output(cli.output);
    let result = dispatch(cli, format);
    std::process::exit(result);
}

fn dispatch(cli: Cli, format: OutputFormat) -> i32 {
    let current_dir = match std::env::current_dir() {
        Ok(path) => path,
        Err(err) => {
            let failure = CommandError::new("harnesscli", "cwd_failed", err.to_string());
            let _ = util::emit_error(&failure, format);
            return 1;
        }
    };

    let outcome = match cli.command {
        Commands::Init => cmd::init::run(&current_dir),
        Commands::Boot(args) => match args.command {
            BootCommand::Start => cmd::boot::start(&current_dir),
            BootCommand::Status => cmd::boot::status(&current_dir),
            BootCommand::Stop => cmd::boot::stop(&current_dir),
        },
        Commands::Smoke => cmd::smoke::run(&current_dir),
        Commands::Test => cmd::test::run(&current_dir),
        Commands::Lint => cmd::lint::run(&current_dir),
        Commands::Typecheck => cmd::typecheck::run(&current_dir),
        Commands::Audit { path } => cmd::audit::run(path.unwrap_or(current_dir)),
        Commands::Cleanup(args) => match args.command {
            CleanupCommand::Scan => cmd::cleanup::scan(&current_dir),
            CleanupCommand::Grade => cmd::cleanup::grade(&current_dir),
            CleanupCommand::Fix => cmd::cleanup::fix(&current_dir),
        },
        Commands::Observability(args) => match args.command {
            ObservabilityCommand::Start => cmd::observability::start(&current_dir),
            ObservabilityCommand::Stop => cmd::observability::stop(&current_dir),
            ObservabilityCommand::Query { kind, query } => {
                cmd::observability::query(&current_dir, &kind, query.as_deref())
            }
        },
    };

    match outcome {
        Ok(bundle) => {
            let _ = util::emit(bundle, format);
            0
        }
        Err(err) => {
            let _ = util::emit_error(&err, format);
            1
        }
    }
}
