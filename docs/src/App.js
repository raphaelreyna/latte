import React from "react";
import AceEditor from "react-ace";
import { Document, Page } from "react-pdf";
import { Base64 } from 'js-base64';

import "ace-builds/src-noconflict/mode-latex";
import "ace-builds/src-noconflict/mode-json";
import "ace-builds/src-noconflict/theme-monokai";

const latteHost = 'https://latte-pdf.herokuapp.com/generate';
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

	componentDidMount() {
		this.submit();
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

	clearTex() {
		this.setState({ tex: '' });
	}

	clearJSON() {
		this.setState({ json: '' });
	}

    render() {
        const pdf = this.state.pdf;
        var pdfURL = undefined;
        if (pdf != undefined) {
            pdfURL = window.URL.createObjectURL(pdf);
        }
		return (
		<div>
			<div id="button-banner">
				<button className="button" onClick={() => this.submit()}>
					Submit
				</button>
				<button className="button" onClick={() => this.clearTex()}>
					Clear LaTeX
				</button>
				<button className="button" onClick={() => this.clearJSON()}>
					Clear JSON
				</button>
        { pdfURL ? <a className="linkButton button" href={pdfURL} download="LaTTe.pdf">Download PDF</a> : null }
				<form id="github" action="https://github.com/raphaelreyna/latte">
    			<input className="button" type="submit" value="Github" />
				</form>
			</div>
			<div className="App">
				<div className="editor">
					<AceEditor
						mode="latex"
						theme="monokai"
						width="85%"
						height="58vh"
						onChange={v => this.setState({ tex: v })}
						value={this.state.tex}
						name="tex"
					/>
					<AceEditor
						mode="json"
						theme="monokai"
						width="85%"
						height="26vh"
						onChange={v => this.setState({ json: v })}
						value={this.state.json}
						name="json"
					/>
				</div>
					<div className="PDFHolder">
						<Document
							className="PDFViewer"
							file={this.state.pdf}
						>
								<Page size="A4" pageNumber={1}/>
						</Document>
					</div>
			</div>
		</div>
		);
	}
}
