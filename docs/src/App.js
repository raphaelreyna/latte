import React from "react";
import AceEditor from "react-ace";
import { Document, Page } from "react-pdf";
import { Base64 } from 'js-base64';

import "ace-builds/src-noconflict/mode-latex";
import "ace-builds/src-noconflict/mode-json";
import "ace-builds/src-noconflict/theme-monokai";

const latteHost = 'http://35.235.126.220/generate';
const headers = {
	'Content-Type': 'application/json',
	'Access-Control-Allow-Origin': '*',
};

const sampleTex = `\\documentclass{article}
\\title{LaTTe Sample Document}
\\begin{document}
\\maketitle
Hello #! .name !#!
\\end{document}`;
const sampleJSON = `{"name": "world"}`;

export default class App extends React.Component {
	constructor(props) {
		super(props);
		this.state = {
			tex: sampleTex,
			json: sampleJSON,
			pdf: undefined,
		};
	}

	async submit() {
		const { tex, json } = this.state;
		const tmpl = Base64.encode(tex);
		const dtls = JSON.parse(json);
		const request = {
			method: 'POST',
			mode: 'cors',
			headers: headers,
			body: JSON.stringify({ template: tmpl, details: dtls}),
		}
		const blob = await fetch(latteHost, request)
			.then(r => r.blob())
			.catch(err => console.log(err));
		this.setState({ pdf: blob });
	}

	render() {
		return ( 
    		<div className="App">
				<div className="editor">
					<AceEditor
						mode="latex"
						theme="monokai"
						width="100%"
						onChange={v => this.setState({ tex: v })}
						value={this.state.tex}
						name="tex"
					/>
					<AceEditor
						mode="json"
						theme="monokai"
						onChange={v => this.setState({ json: v })}
						value={this.state.json}
						name="json"
					/>
				</div>
				<button onClick={() => this.submit()}>
					Submit
				</button>
				<Document 
					className="PDFViewer"
					file={this.state.pdf}
				>
						<Page pageNumber={1}/>
				</Document>
  			</div>
		);
	}
}
