# 実装計画

## 基盤セットアップ

- [ ] 1. プロジェクト構造とコアパッケージの初期設定
  - `cmd/create.go` ファイルを作成し、Cobra コマンドの基本構造を実装
  - 必要なフラグ（--title, --body, --labels, --priority, --status など）を定義
  - コマンドのヘルプテキストと使用例を設定
  - 基本的なフラグ検証ロジックを実装
  - _Requirements: すべての要件に対する基盤設定_

## Issue 管理パッケージの実装

- [ ] 2. Issue データモデルとインターフェースの定義
  - `pkg/issue/models.go` を作成し、IssueData、Issue、ProjectItem 構造体を定義
  - `pkg/issue/errors.go` を作成し、エラータイプと IssueError 構造体を実装
  - 必要な定数（ErrorType）とエラーハンドリングメソッドを定義
  - モデルの検証メソッドを実装
  - _Requirements: 2.1, 2.3, 7.1_

- [ ] 3. Issue Creator サービスの実装
  - `pkg/issue/creator.go` を作成し、Creator 構造体を定義
  - CreateIssue メソッドを実装（GitHub REST API を使用）
  - AddToProject メソッドを実装（GraphQL API を使用）
  - UpdateFields メソッドを実装（プロジェクトフィールドの更新）
  - _Requirements: 2.1, 2.2, 3.1, 3.2_

- [ ] 4. Issue Creator のユニットテスト作成
  - `pkg/issue/creator_test.go` を作成
  - CreateIssue メソッドのテストケースを実装
  - AddToProject メソッドのテストケースを実装
  - モック API クライアントを使用したテストを作成
  - _Requirements: 2.1, 3.1, 3.2_

## 設定管理の拡張

- [ ] 5. 既存の Config パッケージの拡張
  - `pkg/config/config.go` に LoadConfig メソッドを追加
  - 設定ファイルの検証ロジックを強化
  - 親ディレクトリの再帰的検索機能を実装
  - 設定ファイルが見つからない場合のエラーメッセージを改善
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [ ] 6. Config パッケージのユニットテスト作成
  - `pkg/config/config_test.go` に新しいテストケースを追加
  - 設定ファイルの検索ロジックのテスト
  - 検証エラーのテストケース
  - 不正な設定ファイルのテストケース
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

## プロジェクト統合機能

- [ ] 7. ProjectMetadataManager の実装
  - `pkg/project/metadata.go` を作成
  - プロジェクト ID の取得とキャッシュ機能を実装
  - フィールド ID とオプション ID のマッピング機能を実装
  - メタデータのキャッシュ管理機能を追加
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [ ] 8. プロジェクト統合のテスト作成
  - `pkg/project/metadata_test.go` を作成
  - メタデータ取得のテストケース
  - フィールドマッピングのテストケース
  - キャッシュ機能のテストケース
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

## Create コマンドの実装

- [ ] 9. Create コマンドの基本実装
  - `cmd/create.go` に Execute メソッドを実装
  - 設定ファイルの読み込みと検証を統合
  - 単一 Issue の作成フローを実装
  - プロジェクトへの自動追加機能を実装
  - _Requirements: 1.1, 2.1, 2.2, 3.1, 3.2_

- [ ] 10. フラグとオプションの処理実装
  - CLI フラグの解析と検証ロジックを実装
  - デフォルト値の適用ロジックを実装
  - フラグによる値の上書き機能を実装
  - `gh issue create` 互換フラグのパススルー機能を実装
  - _Requirements: 2.3, 3.3, 3.4_

- [ ] 11. 対話モードの実装
  - `pkg/issue/prompt.go` を作成
  - `--interactive` フラグ使用時のプロンプト機能を実装
  - 複数リポジトリ選択のプロンプトを実装
  - 必須フィールドの対話的入力を実装
  - _Requirements: 2.4, 2.5_

## 出力フォーマット機能

- [ ] 12. OutputFormatter の実装
  - `pkg/output/formatter.go` を作成
  - テーブル形式の出力フォーマッタを実装
  - JSON 形式の出力フォーマッタを実装
  - CSV 形式の出力フォーマッタを実装
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [ ] 13. エラー表示とフィードバック機能
  - エラーメッセージのフォーマット機能を実装
  - 推奨される解決策の表示機能を追加
  - `--quiet` モード時の最小限出力を実装
  - 成功時の Issue URL とメタデータ表示を実装
  - _Requirements: 6.1, 6.4, 6.5, 7.1, 7.2, 7.3, 7.4, 7.5_

## バッチ処理機能

- [ ] 14. BatchProcessor の実装
  - `pkg/issue/batch.go` を作成
  - BatchProcessor 構造体と ProcessFile メソッドを実装
  - YAML/JSON ファイルのパースと検証を実装
  - ProcessIssues メソッドで並列処理を実装（goroutine プール）
  - _Requirements: 4.1, 4.2, 4.3_

- [ ] 15. バッチ処理のエラーハンドリングとレポート
  - バッチエラーの収集と集計機能を実装
  - 失敗した Issue のスキップと継続処理を実装
  - 処理完了後のサマリー表示機能を実装
  - BatchResult 構造体の生成と出力を実装
  - _Requirements: 4.3, 6.5_

- [ ] 16. バッチ処理のユニットテスト作成
  - `pkg/issue/batch_test.go` を作成
  - 正常なバッチファイルの処理テスト
  - エラーを含むバッチファイルの処理テスト
  - 並列処理のテストケース
  - _Requirements: 4.1, 4.2, 4.3_

## テンプレート機能

- [ ] 17. TemplateEngine の実装
  - `pkg/template/engine.go` を作成
  - Template と Variable 構造体を定義
  - LoadTemplate メソッドでテンプレートファイルの読み込みを実装
  - Execute メソッドで変数展開機能を実装
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [ ] 18. テンプレート変数の対話的収集
  - `pkg/template/prompter.go` を作成
  - CollectVariables メソッドを実装
  - 必須変数とオプション変数の処理を実装
  - デフォルト値と選択肢の処理を実装
  - _Requirements: 5.2, 5.4_

- [ ] 19. 標準テンプレートの作成
  - `templates/basic.yml` - 基本的な Issue テンプレート
  - `templates/bug.yml` - バグレポート用テンプレート
  - `templates/feature.yml` - 機能要望用テンプレート
  - テンプレートの検証とテストを実装
  - _Requirements: 5.1, 5.3_

## エラーハンドリングとリトライ機能

- [ ] 20. 包括的なエラーハンドリングの実装
  - API レート制限の検出と待機機能を実装
  - ネットワークエラーの自動リトライ機能を実装（最大3回）
  - 権限エラーの詳細表示機能を実装
  - プロジェクトが見つからない場合の利用可能プロジェクト一覧表示を実装
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

## 統合テスト

- [ ] 21. Create コマンドの統合テスト作成
  - `cmd/create_test.go` を作成
  - 単一 Issue 作成の E2E テスト
  - プロジェクトメタデータ適用のテスト
  - エラーケースの統合テスト
  - _Requirements: 2.1, 3.1, 3.2, 6.1_

- [ ] 22. バッチ処理とテンプレート機能の統合テスト
  - `test/integration/batch_test.go` を作成
  - バッチファイルからの Issue 作成テスト
  - テンプレート実行の統合テスト
  - エラーリカバリーのテスト
  - _Requirements: 4.1, 4.2, 5.1, 5.2_

## 最終統合とリファクタリング

- [ ] 23. コマンドの最終統合と最適化
  - すべてのコンポーネントを Create コマンドに統合
  - パフォーマンスの最適化（キャッシング、並列処理）
  - コードのリファクタリングと重複の削除
  - 最終的な動作確認とバグ修正
  - _Requirements: すべての要件の最終検証_