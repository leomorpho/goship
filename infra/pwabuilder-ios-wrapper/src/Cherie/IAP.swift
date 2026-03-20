//
//  IAP.swift
//  pwa-shell
//
//  Created by GlebKh on 05.10.2023.
//

import StoreKit

struct TransactionInfo: Codable {
    let productID: String
    let transactionID: String
}

@MainActor final class StoreKitAPI: ObservableObject {
    // list of products for promoted purchases
    static let IntentsProducts = ["demo_subscription_auto"]
    
    @Published private(set) var products: [Product] = []
    @Published private(set) var productsJson: String = "[]"
    @Published private(set) var activeTransactions: Set<StoreKit.Transaction> = []
    @Published private(set) var activeTransactionsJson: String = "[]"
    private var updates: Task<Void, Never>?
    private var intents: Task<Void, Never>?
    
    init() {
        updates = Task {
            for await update in StoreKit.Transaction.updates {
                if let transaction = try? update.payloadValue {
                    self.activeTransactions.insert(transaction)
                    await transaction.finish()
                }
            }
        }
        intents = Task {
            await fetchProducts()
            await listenToPurchaseIntents()
        }
    }
    
    deinit {
        updates?.cancel()
        intents?.cancel()
    }
    
    // Fetch products list by id's from WebView
    func fetchProducts(productIDs: [String] = StoreKitAPI.IntentsProducts) async {
        do {
            self.products = try await Product.products(for: productIDs)
            
            // Convert each product representation (Data) to JSON String
            let productJSONStrings: [String] = self.products.compactMap { product in
                guard let jsonString = String(data: product.jsonRepresentation, encoding: .utf8) else {
                    return nil
                }
                return jsonString
            }
            
            self.productsJson = "[\(productJSONStrings.joined(separator: ","))]"
            returnProductsResult(jsonString: self.productsJson)
        } catch {
            self.products = []
            // handle error
        }
    }
    
    func purchaseProduct(productID: String, quantity: Int) async throws {
        guard let product = products.first(where: { $0.id == productID }) else {
            // Product not found.
            throw ProductError.productNotFound
        }
        
        let purchaseOption = Product.PurchaseOption.quantity(quantity)
        let purchaseOptions: Set<Product.PurchaseOption> = [purchaseOption]
        
        do {
            let result = try await product.purchase(options: purchaseOptions)
            switch result {
            case .success(let verificationResult):
                if let transaction = try? verificationResult.payloadValue {
                    self.activeTransactions.insert(transaction)
                    await transaction.finish()
                    returnPurchaseTransaction(jsonString: String(data: transaction.jsonRepresentation, encoding: .utf8)!)
                }
            case .userCancelled:
                throw ProductError.userCanceled
            case .pending:
                throw ProductError.pending
            @unknown default:
                throw ProductError.unknown
            }
        } catch {
            // handle or throw error
            throw error
        }
    }
    
    enum ProductError: Error {
        case productNotFound
        case userCanceled
        case pending
        case unknown
    }
    
    // Get all possible transactions. Don't unclude consumable products
    func fetchActiveTransactions() async {
        var activeTransactions: Set<StoreKit.Transaction> = []
        var jsonRepresentation: [String] = []
        
        for await verificationResult in Transaction.all {
            let transaction = verificationResult.unsafePayloadValue
            activeTransactions.insert(transaction)
            jsonRepresentation.append(String(data: transaction.jsonRepresentation, encoding: .utf8)!)
        }
        
        self.activeTransactions = activeTransactions
        self.activeTransactionsJson = "[\(jsonRepresentation.joined(separator: ","))]"
        
        returnActiveTransactions(jsonString: self.activeTransactionsJson)
    }
    
    // https://developer.apple.com/documentation/storekit/in-app_purchase/testing_promoted_in-app_purchases
    // Not tested
    func listenToPurchaseIntents() async {
        if #available(iOS 16.4, *) {
            for await intent in PurchaseIntent.intents {
                do {
                    try await purchaseProduct(productID: intent.product.id, quantity: 1)
                }
                catch {}
            }
        }
    }
}

// push results to webview

func returnProductsResult(jsonString: String){
    DispatchQueue.main.async(execute: {
        Cherie.webView.evaluateJavaScript("this.dispatchEvent(new CustomEvent('iap-products-result', { detail: '\(jsonString)' }))")
    })
}

func returnPurchaseResult(state: String){
    DispatchQueue.main.async(execute: {
        Cherie.webView.evaluateJavaScript("this.dispatchEvent(new CustomEvent('iap-purchase-result', { detail: '\(state)' }))")
    })
}
func returnPurchaseTransaction(jsonString: String){
    DispatchQueue.main.async(execute: {
        Cherie.webView.evaluateJavaScript("this.dispatchEvent(new CustomEvent('iap-purchase-transaction', { detail: '\(jsonString)' }))")
    })
}

func returnActiveTransactions(jsonString: String){
    DispatchQueue.main.async(execute: {
        Cherie.webView.evaluateJavaScript("this.dispatchEvent(new CustomEvent('iap-transactions-result', { detail: '\(jsonString)' }))")
    })
}
