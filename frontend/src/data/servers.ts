export interface RecommendedServerItem {
  id: string;
  serverId?: string;
  title: string;
  version: string;
  loader: string;
  iconKey: 'house' | 'swords' | 'gear' | 'skull' | 'flask';
  subtitle: string;
  btnText: string;
  bannerUrl: string;
  pingMs: number;
  players: string;
  maxPlayers: string;
  description: string;
  tags: string[];
  serverAddress: string;
  region: string;
  features: {
    title: string;
    desc: string;
    badge: string;
  }[];
}

export const RECOMMENDED_SERVERS: RecommendedServerItem[] = [
  {
    id: 'chilltown',
    serverId: 'chilltown',
    title: 'Chill Town',
    version: 'Forge 1.20.1 (v1.0.0)',
    loader: 'forge',
    iconKey: 'flask',
    subtitle: 'Forge 1.20.1 • Bản 1.0.0 Release',
    btnText: 'CHƠI',
    bannerUrl:
      'https://images.unsplash.com/photo-1500595046743-cd271d694d30?auto=format&fit=crop&w=1400&q=80',
    pingMs: 14,
    players: '0',
    maxPlayers: '20',
    description:
      'Máy chủ sinh tồn ẩm thực kết hợp cơ giới hóa nông nghiệp cao cấp, nạp sẵn hơn 246 mods, hệ thống kinh tế SDM Shop và chuỗi nhiệm vụ chuyên sâu.',
    tags: ['246 mods nạp sẵn', 'SDM Shop Kinh Tế', 'P2P WireGuard', 'Đồ Họa Tối Ưu'],
    serverAddress: '127.0.0.1:25565',
    region: 'VN-HOME LAN / P2P',
    features: [
      {
        title: 'Mod Tự Động',
        desc: 'Đồng bộ 246 server mods & KubeJS',
        badge: 'ĐÃ NẠP SẴN',
      },
      {
        title: 'Ẩm Thực & Kỹ Nghệ',
        desc: "Farmer's Delight, Tạo Tác & Brew",
        badge: 'HẤP DẪN',
      },
      {
        title: 'Cam P2P Tunnel',
        desc: 'Độ trễ thấp qua kết nối P2P',
        badge: 'LOW PING',
      },
      {
        title: 'Tự Động Cập Nhật',
        desc: 'Tự động kiểm tra file & mod mới',
        badge: 'SMART SYNC',
      },
    ],
  },
];
