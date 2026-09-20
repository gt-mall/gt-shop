import { pathToFileURL } from 'node:url';
import path from 'node:path';
import fs from 'node:fs';

const srcDir = path.resolve(process.cwd(), 'src');

// 只补充 Vite 提供的环境元数据；请求仍走生产 request.ts，uni 由用例模拟。
export async function load(url, context, nextLoad) {
  const result = await nextLoad(url, context);
  if (url.startsWith(pathToFileURL(srcDir + path.sep).href) && result.source) {
    const source = result.source.toString();
    if (source.includes('import.meta.env')) {
      return { ...result, source: 'import.meta.env = { DEV: true };\n' + source };
    }
  }
  return result;
}

export async function resolve(specifier, context, nextResolve) {
  if (specifier.startsWith('@/')) {
    const relativePath = specifier.slice(2);
    let resolvedPath = path.resolve(srcDir, relativePath);
    if (!fs.existsSync(resolvedPath)) {
      if (fs.existsSync(resolvedPath + '.ts')) {
        resolvedPath = resolvedPath + '.ts';
      } else if (fs.existsSync(resolvedPath + '.js')) {
        resolvedPath = resolvedPath + '.js';
      } else if (fs.existsSync(path.resolve(resolvedPath, 'index.ts'))) {
        resolvedPath = path.resolve(resolvedPath, 'index.ts');
      }
    } else if (fs.statSync(resolvedPath).isDirectory()) {
      if (fs.existsSync(path.resolve(resolvedPath, 'index.ts'))) {
        resolvedPath = path.resolve(resolvedPath, 'index.ts');
      }
    }
    return nextResolve(pathToFileURL(resolvedPath).href, context);
  }

  // 相对路径缺少 .ts 扩展名自动补全
  if (specifier.startsWith('./') || specifier.startsWith('../')) {
    try {
      return await nextResolve(specifier, context);
    } catch (err) {
      if (err.code === 'ERR_MODULE_NOT_FOUND') {
        const parentDir = path.dirname(new URL(context.parentURL).pathname);
        let candidate = path.resolve(parentDir, specifier);
        if (fs.existsSync(candidate + '.ts')) {
          return nextResolve(pathToFileURL(candidate + '.ts').href, context);
        }
        if (fs.existsSync(path.resolve(candidate, 'index.ts'))) {
          return nextResolve(pathToFileURL(path.resolve(candidate, 'index.ts')).href, context);
        }
      }
      throw err;
    }
  }

  return nextResolve(specifier, context);
}
