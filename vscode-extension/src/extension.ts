import * as vscode from 'vscode';
import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind,
	ExecutableOptions,
	Executable
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: vscode.ExtensionContext) {
	// Get configuration
	const config = vscode.workspace.getConfiguration('frugal-ls');
	const serverPath = config.get<string>('server.path', 'frugal-ls');
	const serverArgs = config.get<string[]>('server.args', []);
	
	// Server options for the language server
	const executable: Executable = {
		command: serverPath,
		args: serverArgs,
		transport: TransportKind.stdio,
	};
	
	const serverOptions: ServerOptions = {
		run: executable,
		debug: executable
	};

	// Options to control the language client
	const clientOptions: LanguageClientOptions = {
		// Register the server for Frugal documents
		documentSelector: [{ scheme: 'file', language: 'frugal' }],
		synchronize: {
			// Notify the server about file changes to '.frugal' files contained in the workspace
			fileEvents: vscode.workspace.createFileSystemWatcher('**/*.frugal')
		},
		// Pass workspace configuration to the server
		initializationOptions: {
			// Any initialization options can go here
		},
		middleware: {
			// Add any middleware here if needed
		}
	};

	// Create the language client and start the client.
	client = new LanguageClient(
		'frugal-ls',
		'Frugal Language Server',
		serverOptions,
		clientOptions
	);

	// Start the client. This will also launch the server
	const disposable = client.start();
	context.subscriptions.push(disposable);

	// Register additional commands if needed
	const restartCommand = vscode.commands.registerCommand('frugal-ls.restart', async () => {
		if (client) {
			await client.stop();
			await client.start();
			vscode.window.showInformationMessage('Frugal Language Server restarted');
		}
	});
	context.subscriptions.push(restartCommand);

	// Show status in the status bar
	client.onReady().then(() => {
		vscode.window.showInformationMessage('Frugal Language Server is ready');
	});

	// Handle configuration changes
	context.subscriptions.push(
		vscode.workspace.onDidChangeConfiguration(event => {
			if (event.affectsConfiguration('frugal-ls')) {
				vscode.window.showWarningMessage(
					'Frugal LSP configuration changed. Restart the server for changes to take effect.',
					'Restart Server'
				).then(selection => {
					if (selection === 'Restart Server') {
						vscode.commands.executeCommand('frugal-ls.restart');
					}
				});
			}
		})
	);
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
}