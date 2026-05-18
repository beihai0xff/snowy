/**
 * Snowy v8 · 课纲映射 / 易错点查询前端层
 *
 * 数据来源：`web/snowy-web/data/curriculum/{physics,biology,chemistry}.json`
 *   编译自 docs/curriculum/pep 下的 YAML 数据（脚本：scripts/build-curriculum.cjs）
 *
 * 暴露：
 *   - lookupByTags(subject, tags) → 课纲条目（按 difficulty / 出现频次排序）
 *   - lookupPitfalls(subject, tags) → 易错点（按 severity 排序）
 *   - normalizedSubject(domain)
 */

export type SubjectKey = 'physics' | 'biology' | 'chemistry';

export interface CurriculumRef {
  subject: SubjectKey;
  version: string;
  book: string;
  book_title: string;
  chapter_id: string;
  chapter_title: string;
  section_id: string;
  section_title: string;
  learning_goals: string[];
  core_formulas: string[];
  difficulty: 'easy' | 'medium' | 'hard';
}

export interface Pitfall {
  id: string;
  knowledge_tags: string[];
  pitfall: string;
  correct: string;
  common_mistake?: string;
  correction_hint?: string;
  severity: 'low' | 'medium' | 'high';
  evidence_section?: string;
}

interface SubjectPayload {
  version: string;
  subject: SubjectKey;
  tag_index: Record<string, CurriculumRef[]>;
  pitfalls: Pitfall[];
}

// 静态 import（Next.js 会按需打包到 client bundle）
import physicsData from '@/data/curriculum/physics.json';
import biologyData from '@/data/curriculum/biology.json';
import chemistryData from '@/data/curriculum/chemistry.json';

const DATASETS: Record<SubjectKey, SubjectPayload> = {
  physics: physicsData as unknown as SubjectPayload,
  biology: biologyData as unknown as SubjectPayload,
  chemistry: chemistryData as unknown as SubjectPayload,
};

const TAG_ALIASES: Record<string, string[]> = {
  // Physics
  平抛运动: ['projectile_motion', 'horizontal_uniform', 'vertical_freefall'],
  抛体运动: ['projectile_motion'],
  运动的合成与分解: ['motion_decomposition', 'vector_addition'],
  匀变速运动: ['uniform_accelerated_motion', 'kinematics'],
  匀变速直线运动: ['uniform_accelerated_motion', 'displacement_time'],
  匀速直线运动: ['uniform_motion', 'velocity', '1d_kinematics_simulation'],
  牛顿第二定律: ['newton_second_law', 'force_acceleration_mass'],
  受力分析: ['force_decomposition', 'newton_second_law'],
  弹簧振子: ['spring_oscillation', 'simple_harmonic', 'hooke_law'],
  简谐运动: ['simple_harmonic', 'spring_oscillation'],
  机械能守恒: ['mechanical_energy_conservation'],
  天体运动: ['orbital_motion', 'gravitation'],
  万有引力: ['gravitation', 'orbital_motion'],
  碰撞: ['momentum_conservation', 'kinetic_energy'],
  动量守恒: ['momentum_conservation'],

  // Biology
  光合作用: ['photosynthesis', 'light_reaction', 'calvin_cycle'],
  光照强度: ['photosynthesis', 'limiting_factor'],
  有机物积累: ['photosynthesis', 'limiting_factor'],
  细胞呼吸: ['respiration', 'cellular_respiration'],
  有氧呼吸: ['aerobic_respiration', 'respiration'],
  酶活性: ['enzyme', 'catalysis'],
  遗传分离定律: ['mendel_law', 'segregation_law', 'monohybrid_cross'],
  孟德尔遗传: ['mendel_law', 'segregation_law'],
  自由组合定律: ['independent_assortment', 'dihybrid_cross'],

  // Chemistry
  氧化还原反应: ['redox_reaction', 'electron_transfer', 'oxidation_state'],
  电子转移: ['electron_transfer', 'redox_reaction'],
  离子反应: ['ionic_reaction', 'ion_equation'],
  酸碱中和: ['acid_base', 'neutralization'],
  中和反应: ['acid_base', 'neutralization'],
  配平: ['chemical_equation', 'conservation'],
  电解水: ['electrolysis'],
};

function expandTags(tags: string[]): string[] {
  const out = new Set<string>();
  for (const raw of tags || []) {
    const tag = String(raw || '').trim();
    if (!tag) continue;
    out.add(tag);
    (TAG_ALIASES[tag] || []).forEach((alias) => out.add(alias));
  }
  return Array.from(out);
}

export function normalizedSubject(domain: string | undefined | null): SubjectKey {
  if (domain === 'biology') return 'biology';
  if (domain === 'chemistry') return 'chemistry';
  return 'physics';
}

/**
 * 按 knowledge_tags 反查课纲条目。
 *  - 去重（同 section_id 仅保留一条）
 *  - 命中数高的优先
 *  - 多个 tag 命中同一条目时 boost
 */
export function lookupByTags(subject: SubjectKey, tags: string[]): CurriculumRef[] {
  const expandedTags = expandTags(tags);
  if (expandedTags.length === 0) return [];
  const ds = DATASETS[subject];
  if (!ds) return [];
  const idx = ds.tag_index || {};
  const hit = new Map<string, { ref: CurriculumRef; score: number }>();
  for (const tag of expandedTags) {
    const refs = idx[tag];
    if (!refs) continue;
    for (const ref of refs) {
      const key = `${ref.book}/${ref.section_id}`;
      const exist = hit.get(key);
      if (exist) exist.score += 1;
      else hit.set(key, { ref, score: 1 });
    }
  }
  return Array.from(hit.values())
    .sort((a, b) => b.score - a.score)
    .map((x) => x.ref);
}

/**
 * 按 knowledge_tags 反查易错点。
 *  - 命中至少一个 tag 的易错点
 *  - severity high → medium → low 排序
 *  - 同 severity 内按命中 tag 数排序
 */
export function lookupPitfalls(subject: SubjectKey, tags: string[]): Pitfall[] {
  const ds = DATASETS[subject];
  if (!ds || !ds.pitfalls) return [];
  const expandedTags = expandTags(tags);
  if (expandedTags.length === 0) return [];
  const sevRank: Record<Pitfall['severity'], number> = { high: 0, medium: 1, low: 2 };
  const tagSet = new Set(expandedTags);
  const matched = ds.pitfalls
    .map((p) => {
      const overlap = p.knowledge_tags.filter((t) => tagSet.has(t)).length;
      return overlap > 0 ? { p, overlap } : null;
    })
    .filter((x): x is { p: Pitfall; overlap: number } => !!x)
    .sort((a, b) => {
      const r = sevRank[a.p.severity] - sevRank[b.p.severity];
      if (r !== 0) return r;
      return b.overlap - a.overlap;
    });
  return matched.map((m) => m.p);
}

/**
 * 取前 N 个最相关条目（默认 1 个 → badge 显示）
 */
export function topCurriculum(subject: SubjectKey, tags: string[], limit = 1): CurriculumRef[] {
  return lookupByTags(subject, tags).slice(0, limit);
}

/**
 * 取前 N 个最严重易错点
 */
export function topPitfalls(subject: SubjectKey, tags: string[], limit = 3): Pitfall[] {
  return lookupPitfalls(subject, tags).slice(0, limit);
}

/** 格式化课纲条目为人可读字符串 */
export function formatCurriculumLabel(ref: CurriculumRef): string {
  const bookShort = ref.book.replace('compulsory-', '必修');
  return `人教版·${bookShort}·§${ref.section_id} ${ref.section_title}`;
}

/** debug：取所有 tag */
export function allTags(subject: SubjectKey): string[] {
  return Object.keys(DATASETS[subject]?.tag_index || {});
}

