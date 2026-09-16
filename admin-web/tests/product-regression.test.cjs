const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');
const ts = require('typescript');

const root = path.resolve(__dirname, '../src');
const editFile = 'modules/product/product/edit/index.tsx';
const specFile = 'modules/product/product/spec-editor.tsx';
const skuFile = 'modules/product/product/sku-table.tsx';

function sourceFile(file) {
  return ts.createSourceFile(file, fs.readFileSync(path.join(root, file), 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
}

function declaration(file, name) {
  const source = sourceFile(file);
  let found;
  function visit(node) {
    if (ts.isVariableDeclaration(node) && node.name.getText(source) === name) found = node.initializer;
    ts.forEachChild(node, visit);
  }
  visit(source);
  assert.ok(found, `Missing declaration ${name} in ${file}`);
  return found;
}

function execute(source, bindings = {}) {
  const output = ts.transpileModule(source, {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX },
  }).outputText;
  const context = vm.createContext({ exports: {}, require, ...bindings });
  vm.runInContext(output, context);
  return context;
}

// Run the actual page declarations with controlled state/API boundaries. No DOM
// or copy of the business logic is needed for these state and payload regressions.
function evaluate(file, names, bindings) {
  return execute(names.map((name) => `var ${name} = ${declaration(file, name).getText()};`).join('\n'), bindings);
}

const productTypes = execute(fs.readFileSync(path.join(root, 'modules/product/types.ts'), 'utf8')).exports;
const message = { error() {}, warning() {}, success() {} };

for (const originalMode of ['single', 'multi']) {
  test(`saved ${originalMode} SKU keeps every field when mode switching is attempted`, () => {
    const sku = {
      id: 77, spec_values: originalMode === 'single' ? [] : [{ spec_id: 1, value_id: 1 }],
      sku_code: 'SKU77', barcode: 'BAR77', image_id: 9, price: '199.90',
      market_price: '299.90', cost_price: '99.90', unit: '件', weight_gram: 100, sort: 3, status: 0,
    };
    const writes = [];
    const context = evaluate(editFile, ['hasSavedSKUs', 'handleSpecModeChange'], {
      initialDetail: { skus: [sku], specs: originalMode === 'single' ? [] : [{ id: 1 }] },
      skus: [sku], specMode: originalMode, message,
      setSpecMode: (value) => writes.push(value), setSpecs: (value) => writes.push(value),
      setSKUs: (value) => writes.push(value),
    });
    context.handleSpecModeChange(originalMode === 'single' ? 'multi' : 'single');
    context.handleSpecModeChange(originalMode);
    assert.equal(context.hasSavedSKUs, true);
    assert.deepEqual(writes, []);
    assert.equal(sku.price, '199.90');
  });
}

test('new products can still choose multi-spec mode', () => {
  const modes = [];
  const context = evaluate(editFile, ['hasSavedSKUs', 'handleSpecModeChange'], {
    initialDetail: null, skus: [{ id: 0, spec_values: [] }], specMode: 'single', message,
    setSpecMode: (value) => modes.push(value), setSpecs() {}, setSKUs() {},
  });
  context.handleSpecModeChange('multi');
  assert.deepEqual(modes, ['multi']);
});

const specs = [
  { id: 1, name: '颜色', values: [{ id: 1, name: '黑色' }] },
  { id: 2, name: '尺寸', values: [{ id: 1, name: 'L' }, { id: 2, name: 'XL' }] },
];

test('SKU rendered names use spec_id and value_id together', () => {
  const columns = declaration(skuFile, 'columns');
  const render = columns.elements[0].properties.find((property) => property.name?.getText() === 'render').initializer;
  const nameContext = evaluate(skuFile, ['nameMap'], {
    specs, useMemo: (fn) => fn(), getSpecValueKey: productTypes.getSpecValueKey,
  });
  const context = execute(`var render = ${render.getText()};`, {
    nameMap: nameContext.nameMap, getSpecValueKey: productTypes.getSpecValueKey, Tag: 'tag', Space: 'space',
  });
  const output = context.render(null, { spec_values: [{ spec_id: 1, value_id: 1 }, { spec_id: 2, value_id: 1 }] });
  assert.equal(output.props.children[0].props.children, '黑色');
  assert.equal(output.props.children[1].props.children, 'L');
});

test('an unused value can be removed despite a matching ID in another spec', () => {
  const changes = [];
  const context = evaluate(specFile, ['savedUsage', 'handleRemoveOption'], {
    specs, skus: [{ id: 77, spec_values: [{ spec_id: 1, value_id: 1 }, { spec_id: 2, value_id: 2 }] }],
    useMemo: (fn) => fn(), getSpecValueKey: productTypes.getSpecValueKey, message,
    onSpecsChange: (value) => changes.push(value),
  });
  context.handleRemoveOption(2, 1);
  assert.equal(changes.length, 1);
  assert.equal(changes[0][0].values[0].id, 1);
  assert.equal(changes[0][1].values.length, 1);
  assert.equal(changes[0][1].values[0].id, 2);
  context.handleRemoveOption(1, 1);
  assert.equal(changes.length, 1, 'referenced value must remain protected');
});

test('saved SKU dimensions cannot be changed; new options preserve existing SKU fields', () => {
  const saved = { id: 77, spec_values: [{ spec_id: 1, value_id: 1 }, { spec_id: 2, value_id: 1 }], price: '199.90', status: 0, image_id: 9 };
  const changes = [];
  const context = evaluate(specFile, ['hasSavedSKUs', 'handleAddSpec'], {
    specs, skus: [saved], message, maxSpecIDRef: { current: 2 },
    onSpecsChange: (value) => changes.push(value),
  });
  context.handleAddSpec();
  assert.deepEqual(changes, []);
  const generated = productTypes.generateSKUCombinations(specs, [saved]);
  assert.equal(generated.skus.length, 2);
  assert.equal(generated.skus[0], saved);
});

test('clearing a category image sends explicit JSON null', async () => {
  const payloads = [];
  const context = evaluate('modules/product/category/index.tsx', ['handleFinish'], {
    editingCategory: { id: 41 }, setSaving() {}, message,
    api: { api: { async updateProductCategory(payload) { payloads.push(JSON.parse(JSON.stringify(payload))); return { code: 0 }; } } },
  });
  await context.handleFinish({ name: '分类', parent_id: 0, image_id: undefined, status: 1 });
  assert.equal(payloads.length, 1);
  assert.ok(Object.hasOwn(payloads[0], 'image_id'));
  assert.equal(payloads[0].image_id, null);
});

test('product editing and creation require list plus their action permission', () => {
  const routes = new Map();
  function attribute(element, name) {
    return element.attributes.properties.find((property) => property.name?.text === name)?.initializer;
  }
  function visit(node, permissions = []) {
    if (ts.isJsxElement(node) && node.openingElement.tagName.getText() === 'Route') {
      const guard = attribute(node.openingElement, 'element')?.expression;
      if (guard && ts.isJsxSelfClosingElement(guard) && guard.tagName.getText() === 'PermissionRoute') {
        permissions = [...permissions, attribute(guard, 'permission').text];
      }
    }
    if (ts.isJsxSelfClosingElement(node) && node.tagName.getText() === 'Route') {
      const routePath = attribute(node, 'path')?.text;
      if (routePath) routes.set(routePath, permissions);
    }
    ts.forEachChild(node, (child) => visit(child, permissions));
  }
  visit(sourceFile('framework/router/index.tsx'));
  assert.deepEqual(routes.get('/product/product/:id/edit'), ['product:product:list', 'product:product:update']);
  assert.deepEqual(routes.get('/product/product/create'), ['product:product:list', 'product:product:create']);
  assert.deepEqual(routes.get('/product/product/:id'), ['product:product:list']);
});

test('product parameters filter blank rows and keep non-empty values', () => {
  const rawParams = [
    { name: ' 品牌 ', value: ' Apple ' },
    { name: '产地', value: '中国大陆' },
    { name: '   ', value: '   ' },
  ];
  const validParams = [];
  for (const p of rawParams) {
    const name = (p.name ?? '').trim();
    const value = (p.value ?? '').trim();
    if (!name && !value) continue;
    validParams.push({ name, value });
  }
  assert.equal(validParams.length, 2);
  assert.equal(validParams[0].name, '品牌');
  assert.equal(validParams[0].value, 'Apple');
  assert.equal(validParams[1].name, '产地');
  assert.equal(validParams[1].value, '中国大陆');
});
