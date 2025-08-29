# 実装計画

## 基盤となる設定とメタデータ構造

- [ ] 1. Config構造体にメタデータサポートを追加
  - pkg/config/config.go に ConfigMetadata、ProjectMetadata、FieldsMetadata、FieldMetadata 構造体を追加
  - Config構造体に Metadata フィールドを追加（omitempty タグ付き）
  - SaveWithMetadata メソッドを実装して、メタデータ付きでYAML保存可能にする
  - LoadMetadata メソッドを実装して、メタデータの読み込みを可能にする
  - 既存の Save メソッドが後方互換性を保つことを確認
  - _Requirements: 8.5, 7.5_

- [ ] 2. メタデータ構造体のユニットテストを作成
  - pkg/config/config_test.go に SaveWithMetadata のテストを追加
  - メタデータあり/なしの両方のケースをテスト
  - YAML形式が正しく出力されることを検証
  - 既存設定ファイルとの後方互換性をテスト
  - _Requirements: 8.7_

## プロジェクト検出とメタデータ管理

- [ ] 3. ProjectDetector パッケージを実装
  - pkg/init/detector.go を新規作成
  - ProjectDetector 構造体と NewProjectDetector コンストラクタを実装
  - DetectCurrentRepo メソッドを実装（repository.Current() を使用）
  - ListRepoProjects と ListOrgProjects メソッドを実装（project.Client を使用）
  - _Requirements: 2.1, 2.2, 2.3_

- [ ] 4. MetadataManager パッケージを実装
  - pkg/init/metadata.go を新規作成
  - MetadataManager 構造体と NewMetadataManager コンストラクタを実装
  - FetchProjectMetadata メソッドを実装（プロジェクトnode ID取得）
  - FetchFieldMetadata メソッドを実装（フィールドIDとオプションID取得）
  - BuildMetadata メソッドを実装（完全なメタデータ構造を構築）
  - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [ ] 5. GraphQL クエリの拡張
  - pkg/project/project.go に GetProjectNodeID メソッドを追加
  - GetFieldsWithOptions メソッドを追加（フィールドIDとオプションIDを含む）
  - GraphQLクエリにnode IDフィールドを追加
  - オプションIDを取得するためのネストされたクエリを実装
  - _Requirements: 6.1, 8.1, 8.2_

## 対話的インターフェース

- [ ] 6. InteractivePrompt パッケージを実装
  - pkg/init/prompt.go を新規作成
  - InteractivePrompt 構造体と NewInteractivePrompt コンストラクタを実装
  - ConfirmOverwrite メソッドを実装（上書き確認）
  - SelectProject メソッドを実装（プロジェクト選択UI）
  - GetStringInput メソッドを実装（汎用文字列入力）
  - _Requirements: 1.2, 4.1, 4.3, 4.4_

- [ ] 7. フィールドマッピング設定UIを実装
  - InteractivePrompt に ConfigureFieldMapping メソッドを追加
  - 自動マッピング候補の生成ロジックを実装
  - ユーザーによるカスタマイズ機能を実装
  - ステータスと優先度の両フィールドに対応
  - _Requirements: 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9, 5.10, 5.11, 5.12, 5.13, 5.14_

## initコマンドの統合

- [ ] 8. initコマンドに --skip-metadata フラグを追加
  - cmd/init.go に skipMetadata 変数を追加
  - initCmd.Flags() に --skip-metadata フラグを登録
  - フラグの説明を追加（「メタデータ取得をスキップ」）
  - _Requirements: 8.6_

- [ ] 9. initコマンドにメタデータ取得ロジックを統合
  - runInit 関数で MetadataManager を初期化
  - プロジェクト選択後にメタデータを取得
  - フィールドマッピング設定後にオプションIDを取得
  - --skip-metadata フラグが指定された場合は処理をスキップ
  - エラー発生時も基本設定は作成するようにエラーハンドリング
  - _Requirements: 8.1, 8.6, 8.7_

- [ ] 10. プロジェクト自動検出ロジックを統合
  - runInit 関数で ProjectDetector を使用
  - 現在のリポジトリを検出してプロジェクト一覧を取得
  - リポジトリにプロジェクトがない場合は組織プロジェクトを取得
  - selectProjectWithDetails 関数を使用してプロジェクト選択
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8_

- [ ] 11. コマンドラインフラグの検証と処理
  - --project フラグの処理を確認
  - --org フラグの処理を確認  
  - --repo フラグの複数指定対応を確認
  - --interactive=false の場合の非対話的処理を確認
  - --list フラグで全プロジェクト表示を確認
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_

## テストとエラーハンドリング

- [ ] 12. ProjectDetector のユニットテストを作成
  - pkg/init/detector_test.go を新規作成
  - モックのGitHub APIクライアントを使用
  - リポジトリ検出のテスト
  - プロジェクト一覧取得のテスト
  - エラーケースのテスト
  - _Requirements: 2.1, 2.2, 2.3_

- [ ] 13. MetadataManager のユニットテストを作成
  - pkg/init/metadata_test.go を新規作成
  - モックのproject.Client を使用
  - メタデータ取得のテスト
  - フィールドIDとオプションIDマッピングのテスト
  - 部分的な成功ケースのテスト
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.7_

- [ ] 14. InteractivePrompt のユニットテストを作成
  - pkg/init/prompt_test.go を新規作成
  - 入力のモックとテスト
  - プロジェクト選択ロジックのテスト
  - フィールドマッピング設定のテスト
  - デフォルト値処理のテスト
  - _Requirements: 4.2, 4.5, 4.6, 5.3_

- [ ] 15. initコマンドの統合テストを作成
  - cmd/init_test.go を拡張
  - 対話的モードの統合テスト
  - 非対話的モード（フラグ指定）の統合テスト
  - メタデータ付き設定ファイル生成のテスト
  - --skip-metadata フラグのテスト
  - 既存設定ファイルの上書きテスト
  - _Requirements: 1.1, 1.3, 1.4, 1.5, 3.5, 8.6_

- [ ] 16. エラーハンドリングの実装と検証
  - InitError 型を pkg/init/errors.go に実装
  - handleInitError 関数を実装
  - GitHub API エラーの適切な処理を確認
  - ファイルシステムエラーの処理を確認
  - メタデータ取得失敗時の継続処理を確認
  - _Requirements: 6.4, 6.5, 8.7_

## デフォルト値と最終検証

- [ ] 17. デフォルト値の設定を検証
  - DefaultConfig 関数のデフォルト値を確認
  - priority: "medium"、status: "Todo" の設定を確認
  - "pm-tracked" ラベルの設定を確認
  - 現在のリポジトリをリストの先頭に配置する処理を確認
  - output format: "table" の設定を確認
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

- [ ] 18. プロジェクト詳細の自動取得を検証
  - プロジェクト番号の自動取得を確認
  - カスタムフィールド一覧の表示を確認
  - API接続エラー時の警告表示を確認
  - プロジェクトが見つからない場合の処理を確認
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [ ] 19. 全要件のEnd-to-Endテストを作成
  - 実際のGitHub APIを使用した統合テスト（CIでのみ実行）
  - 対話的フローの完全なシナリオテスト
  - メタデータ保存と読み込みの検証
  - 設定ファイルの形式と内容の検証
  - _Requirements: 全要件のE2E検証_