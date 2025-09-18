// Load WASM
async function loadWasm() {
    if (!WebAssembly.instantiateStreaming) {
        WebAssembly.instantiateStreaming = async (resp, importObject) => {
            const source = await (await resp).arrayBuffer();
            return await WebAssembly.instantiate(source, importObject);
        };
    }

    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(fetch("xmlq.wasm"), go.importObject);
    go.run(result.instance);
}

loadWasm();

// Add mask field dynamically
document.getElementById("add-mask").addEventListener("click", () => {
    const masksContainer = document.getElementById("masks");
    const maskDiv = document.createElement("div");
    maskDiv.className = "mask";
    maskDiv.innerHTML = `
        <input type="text" class="mask-name" placeholder="Element Name" />
        <input type="text" class="mask-space" placeholder="Namespace (optional)" />
        <select class="mask-type">
            <option value="show-last-four">Show Last Four</option>
            <option value="show-middle">Show Middle</option>
            <option value="show-word-start">Show Word Start</option>
            <option value="show-none">Show None</option>
        </select>
        <button class="remove-mask">Remove</button>
    `;
    masksContainer.appendChild(maskDiv);

    // Add event listener to the new remove button
    maskDiv.querySelector(".remove-mask").addEventListener("click", () => {
        maskDiv.remove();
    });
});

// Remove mask field
document.querySelectorAll(".remove-mask").forEach(button => {
    button.addEventListener("click", () => {
        button.parentElement.remove();
    });
});

// Process XML
document.getElementById("process").addEventListener("click", async () => {
    // Read input XML from textarea
    const inputXML = document.getElementById("input-xml").value;

    // Read prefix and indent from input fields
    const prefix = document.getElementById("prefix").value;
    const indent = document.getElementById("indent").value;

    // Read masks from form elements
    const masks = [];
    document.querySelectorAll(".mask").forEach(maskDiv => {
        const name = maskDiv.querySelector(".mask-name").value;
        const space = maskDiv.querySelector(".mask-space").value;
        const type = maskDiv.querySelector(".mask-type").value;
        if (name) { // Only include masks with a name
            masks.push({
                name: name,
                space: space,
                mask: type,
            });
        }
    });

    // Construct options object
    const options = {
        prefix: prefix,
        indent: indent,
        masks: masks,
    };

    // Call WASM function
    const result = maskXML(inputXML, options);
    const output = document.getElementById("output-xml");
    if (result.error) {
        output.value = `Error: ${result.error}`;
    } else {
        output.value = result.result;
    }
});
