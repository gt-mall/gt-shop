import { create } from 'zustand';

export type AppStore = {
  activeModule: string;
  selectedMenu: string;
  selectModule: (moduleKey: string, firstMenuKey: string) => void;
  selectMenu: (menuKey: string) => void;
};

export const useAppStore = create<AppStore>((set) => ({
  activeModule: 'home',
  selectedMenu: 'home-subnav1-option1',
  selectModule: (moduleKey, firstMenuKey) => set({ activeModule: moduleKey, selectedMenu: firstMenuKey }),
  selectMenu: (menuKey) => set({ selectedMenu: menuKey }),
}));
